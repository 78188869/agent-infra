package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/example/agent-infra/internal/api/router"
	"github.com/example/agent-infra/internal/config"
	"github.com/example/agent-infra/internal/executor"
	"github.com/example/agent-infra/internal/migration"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/monitoring"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/internal/scheduler"
	"github.com/example/agent-infra/internal/seed"
	"github.com/example/agent-infra/internal/service"
	"github.com/example/agent-infra/pkg/aliyun/sls"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load config (respects APP_ENV=local for config.local.yaml)
	configPath := config.ResolveConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	env := cfg.GetEnvironment()
	logger := monitoring.NewLogger(cfg)
	slog.SetDefault(logger)
	slog.Info("starting control-plane", "env", env, "config", configPath)
	gin.SetMode(cfg.Server.Mode)

	// 2. Ensure data directory exists for SQLite
	if cfg.Database.IsSQLite() {
		dbName := cfg.Database.Database
		if dbName == "" {
			dbName = "agent_infra.db"
		}
		dir := filepath.Dir(dbName)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				slog.Error("failed to create data directory", "dir", dir, "error", err)
				os.Exit(1)
			}
		}
	}

	// 3. Database
	db, err := config.NewDatabase(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Warn("error closing database", "error", err)
		}
	}()

	// 4. Migrations + seed
	m := migration.NewMigrator(db.DB)
	if err := m.AutoMigrate(); err != nil {
		slog.Warn("auto-migrate failed", "error", err)
	}
	if cfg.IsLocal() {
		if err := seed.SeedProviders(db.DB); err != nil {
			slog.Warn("seed providers failed", "error", err)
		}
	}

	// 5. Redis (miniredis for local, real Redis for production)
	var redisClient *redis.Client
	var miniRedis *miniredis.Miniredis
	if cfg.IsLocal() {
		miniRedis = miniredis.NewMiniRedis()
		miniRedis.Start()
		redisClient = redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
		slog.Info("using miniredis for local development", "addr", miniRedis.Addr())
	} else {
		redisCfg := cfg.Redis.ToRedisConfig()
		rc, err := config.NewRedisClient(redisCfg)
		if err != nil {
			slog.Error("failed to connect to Redis", "error", err)
			os.Exit(1)
		}
		redisClient = rc.Client
	}

	// 6. Repositories
	tenantRepo := repository.NewTenantRepository(db.DB)
	templateRepo := repository.NewTemplateRepository(db.DB)
	taskRepo := repository.NewTaskRepository(db.DB)
	providerRepo := repository.NewProviderRepository(db.DB)
	capabilityRepo := repository.NewCapabilityRepository(db.DB)
	interventionRepo := repository.NewInterventionRepository(db.DB)

	// 7. Services (handler -> service -> repository -> model)
	tenantSvc := service.NewTenantService(tenantRepo)
	templateSvc := service.NewTemplateService(templateRepo)
	taskSvc := service.NewTaskService(taskRepo)
	providerSvc := service.NewProviderService(providerRepo)
	capabilitySvc := service.NewCapabilityService(capabilityRepo)
	interventionSvc := service.NewInterventionService(taskRepo, interventionRepo)

	// 8. Monitoring
	monitoringHub := monitoring.NewHub()
	slsClient := monitoring.NewSLSClient(sls.Config{
		Endpoint:        cfg.SLS.Endpoint,
		AccessKeyID:     cfg.SLS.AccessKey,
		AccessKeySecret: cfg.SLS.AccessSecret,
		Project:         cfg.SLS.Project,
		LogStore:        cfg.SLS.Logstore,
	})
	monitoringSvc := service.NewMonitoringService(monitoringHub, slsClient)

	// 9. Executor (Docker runtime for local, optional)
	var taskExec *executor.TaskExecutor
	if cfg.IsLocal() {
		dockerRuntime, err := executor.NewDockerRuntime(executor.DefaultDockerConfig())
		if err != nil {
			slog.Warn("Docker runtime not available, executor disabled", "error", err)
		} else {
			taskExec, err = executor.NewTaskExecutor(dockerRuntime, redisClient, &executor.ExecutorConfig{
				// IMPORTANT: Use taskSvc.UpdateStatus, NOT taskRepo.UpdateStatus
				// Architecture constraint: executor -> service -> repo
				UpdateTaskStatus: func(ctx context.Context, taskID, status, message string) error {
					return taskSvc.UpdateStatus(ctx, taskID, status, message)
				},
				GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
					return taskSvc.GetByID(ctx, taskID)
				},
				OnTaskComplete: func(ctx context.Context, taskID string, result map[string]interface{}) error {
					return taskSvc.UpdateStatus(ctx, taskID, model.TaskStatusSucceeded, "task completed")
				},
				OnTaskFailed: func(ctx context.Context, taskID string, taskErr error) error {
					return taskSvc.UpdateStatus(ctx, taskID, model.TaskStatusFailed, taskErr.Error())
				},
			})
			if err != nil {
				slog.Warn("failed to create task executor", "error", err)
				taskExec = nil
			} else {
				// Wire executor as BOTH event handler AND instruction injector
				service.SetInterventionEventHandler(interventionSvc, taskExec)
				service.SetInterventionInjector(interventionSvc, taskExec)
			}
		}
	}

	// 10. Scheduler
	sched := scheduler.NewTaskScheduler(redisClient, &scheduler.SchedulerConfig{
		GlobalLimit: 100,
		GetTenantQuota: func(ctx context.Context, tenantID string) (*scheduler.TenantQuota, error) {
			tenant, err := tenantSvc.GetByID(ctx, tenantID)
			if err != nil {
				return nil, err
			}
			return &scheduler.TenantQuota{
				Concurrency: tenant.QuotaConcurrency,
				DailyTasks:  tenant.QuotaDailyTasks,
			}, nil
		},
		GetTask: func(ctx context.Context, taskID string) (*model.Task, error) {
			return taskSvc.GetByID(ctx, taskID)
		},
		// IMPORTANT: Use taskSvc.UpdateStatus for architecture compliance
		UpdateStatus: func(ctx context.Context, taskID, status, message string) error {
			return taskSvc.UpdateStatus(ctx, taskID, status, message)
		},
	})

	// 11. Start executor + scheduler
	ctx := context.Background()
	if taskExec != nil {
		if err := taskExec.Start(ctx); err != nil {
			slog.Warn("executor start failed", "error", err)
		}
	}
	if err := sched.Start(ctx); err != nil {
		slog.Warn("scheduler start failed", "error", err)
	}

	// 12. Router
	r := router.Setup(tenantSvc, templateSvc, taskSvc, providerSvc, capabilitySvc, monitoringSvc, monitoringHub, interventionSvc, db)

	// 13. HTTP server with graceful shutdown
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		slog.Info("starting HTTP server", "addr", addr, "env", env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig)

	// Shutdown in reverse order of startup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sched.Stop(shutdownCtx); err != nil {
		slog.Warn("scheduler stop error", "error", err)
	}
	if taskExec != nil {
		if err := taskExec.Stop(shutdownCtx); err != nil {
			slog.Warn("executor stop error", "error", err)
		}
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("HTTP server shutdown error", "error", err)
	}
	if miniRedis != nil {
		miniRedis.Close()
	}
	if err := db.Close(); err != nil {
		slog.Warn("database close error", "error", err)
	}

	slog.Info("server exited")
}
