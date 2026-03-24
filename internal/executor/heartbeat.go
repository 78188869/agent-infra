package executor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	// HeartbeatKeyPattern is the Redis key pattern for task heartbeats.
	HeartbeatKeyPattern = "executor:heartbeat:%s"

	// DefaultHeartbeatInterval is the default interval between heartbeats.
	DefaultHeartbeatInterval = 5 * time.Second

	// DefaultHeartbeatTimeout is the default timeout for missing heartbeats.
	DefaultHeartbeatTimeout = 15 * time.Second
)

// HeartbeatManager manages heartbeat detection for running tasks.
type HeartbeatManager struct {
	client   RedisClient
	interval time.Duration
	timeout  time.Duration

	// Tracking running tasks
	mu       sync.RWMutex
	tasks    map[string]*HeartbeatInfo
	stopCh   chan struct{}
	running  bool
	wg       sync.WaitGroup
	stopOnce sync.Once

	// Callbacks
	onTimeout func(ctx context.Context, taskID string) error

	// Error channel for reporting callback errors (optional)
	errorChan chan error
}

// HeartbeatInfo contains heartbeat information for a task.
type HeartbeatInfo struct {
	TaskID    string
	LastSeen  time.Time
	Status    string
	Progress  int
	PodIP     string
	StartedAt time.Time
}

// HeartbeatManagerConfig holds configuration for HeartbeatManager.
type HeartbeatManagerConfig struct {
	Interval  time.Duration
	Timeout   time.Duration
	OnTimeout func(ctx context.Context, taskID string) error

	// ErrorChanSize specifies the buffer size for the error channel.
	// If 0 (default), no error channel is created.
	// Set to a positive value to enable error reporting via channel.
	ErrorChanSize int
}

// NewHeartbeatManager creates a new HeartbeatManager instance.
func NewHeartbeatManager(client RedisClient, cfg *HeartbeatManagerConfig) (*HeartbeatManager, error) {
	if client == nil {
		return nil, ErrNilRedisClient
	}

	if cfg == nil {
		cfg = &HeartbeatManagerConfig{}
	}
	if cfg.Interval == 0 {
		cfg.Interval = DefaultHeartbeatInterval
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultHeartbeatTimeout
	}

	hm := &HeartbeatManager{
		client:    client,
		interval:  cfg.Interval,
		timeout:   cfg.Timeout,
		tasks:     make(map[string]*HeartbeatInfo),
		stopCh:    make(chan struct{}),
		onTimeout: cfg.OnTimeout,
	}

	// Create error channel if configured
	if cfg.ErrorChanSize > 0 {
		hm.errorChan = make(chan error, cfg.ErrorChanSize)
	}

	return hm, nil
}

// Register registers a task for heartbeat monitoring.
func (m *HeartbeatManager) Register(taskID string, podIP string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks[taskID] = &HeartbeatInfo{
		TaskID:    taskID,
		LastSeen:  time.Now(),
		Status:    "running",
		PodIP:     podIP,
		StartedAt: time.Now(),
	}
}

// Unregister removes a task from heartbeat monitoring.
func (m *HeartbeatManager) Unregister(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tasks, taskID)
}

// UpdateHeartbeat updates the heartbeat timestamp for a task.
func (m *HeartbeatManager) UpdateHeartbeat(ctx context.Context, taskID string, status string, progress int) error {
	key := fmt.Sprintf(HeartbeatKeyPattern, taskID)

	// Update Redis
	pipe := m.client.Pipeline()
	pipe.HSet(ctx, key, "last_seen", time.Now().Unix())
	pipe.HSet(ctx, key, "status", status)
	pipe.HSet(ctx, key, "progress", progress)
	pipe.Expire(ctx, key, m.timeout*2) // Set expiration to 2x timeout

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	// Update local tracking
	m.mu.Lock()
	if info, exists := m.tasks[taskID]; exists {
		info.LastSeen = time.Now()
		info.Status = status
		info.Progress = progress
	}
	m.mu.Unlock()

	return nil
}

// GetHeartbeat retrieves the heartbeat info for a task.
func (m *HeartbeatManager) GetHeartbeat(ctx context.Context, taskID string) (*HeartbeatInfo, error) {
	key := fmt.Sprintf(HeartbeatKeyPattern, taskID)

	result, err := m.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeat: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // No heartbeat found
	}

	info := &HeartbeatInfo{
		TaskID: taskID,
	}

	if v, ok := result["last_seen"]; ok {
		var timestamp int64
		fmt.Sscanf(v, "%d", &timestamp)
		info.LastSeen = time.Unix(timestamp, 0)
	}

	if v, ok := result["status"]; ok {
		info.Status = v
	}

	if v, ok := result["progress"]; ok {
		fmt.Sscanf(v, "%d", &info.Progress)
	}

	return info, nil
}

// Start begins the heartbeat monitoring loop.
func (m *HeartbeatManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return ErrExecutorAlreadyRunning
	}
	m.running = true
	m.mu.Unlock()

	m.wg.Add(1)
	go m.monitorLoop(ctx)

	return nil
}

// Stop stops the heartbeat monitoring.
func (m *HeartbeatManager) Stop(ctx context.Context) error {
	var err error
	m.stopOnce.Do(func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()

		close(m.stopCh)

		// Wait for monitor loop to stop
		done := make(chan struct{})
		go func() {
			m.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}

// IsRunning returns whether the heartbeat manager is running.
func (m *HeartbeatManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetTaskCount returns the number of tasks being monitored.
func (m *HeartbeatManager) GetTaskCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tasks)
}

// Errors returns a channel for receiving callback errors.
// Returns nil if error channel is not configured (ErrorChanSize = 0).
// The channel is buffered with the size specified in HeartbeatManagerConfig.
// Errors sent to this channel are from OnTimeout callback failures.
func (m *HeartbeatManager) Errors() <-chan error {
	return m.errorChan
}

// monitorLoop runs the periodic heartbeat check.
func (m *HeartbeatManager) monitorLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkHeartbeats(ctx)
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkHeartbeats checks all registered tasks for heartbeat timeouts.
func (m *HeartbeatManager) checkHeartbeats(ctx context.Context) {
	m.mu.RLock()
	taskIDs := make([]string, 0, len(m.tasks))
	for id := range m.tasks {
		taskIDs = append(taskIDs, id)
	}
	m.mu.RUnlock()

	now := time.Now()

	for _, taskID := range taskIDs {
		// Get heartbeat from Redis
		info, err := m.GetHeartbeat(ctx, taskID)
		if err != nil {
			continue // Skip on error
		}

		if info == nil {
			// No heartbeat in Redis, check local
			m.mu.RLock()
			localInfo, exists := m.tasks[taskID]
			m.mu.RUnlock()

			if !exists {
				continue
			}

			if now.Sub(localInfo.LastSeen) > m.timeout {
				m.handleTimeout(ctx, taskID)
			}
			continue
		}

		// Check timeout
		if now.Sub(info.LastSeen) > m.timeout {
			m.handleTimeout(ctx, taskID)
		}
	}
}

// handleTimeout handles a heartbeat timeout event.
func (m *HeartbeatManager) handleTimeout(ctx context.Context, taskID string) {
	// Remove from tracking
	m.mu.Lock()
	delete(m.tasks, taskID)
	m.mu.Unlock()

	// Clear Redis key
	key := fmt.Sprintf(HeartbeatKeyPattern, taskID)
	m.client.Del(ctx, key)

	// Call timeout callback
	if m.onTimeout != nil {
		if err := m.onTimeout(ctx, taskID); err != nil {
			// Log the error - this is important for debugging task state inconsistencies
			log.Printf("[HeartbeatManager] OnTimeout callback failed for task %s: %v", taskID, err)

			// Send error to channel if available (non-blocking)
			if m.errorChan != nil {
				select {
				case m.errorChan <- fmt.Errorf("OnTimeout callback failed for task %s: %w", taskID, err):
				default:
					// Channel full, log and skip
					log.Printf("[HeartbeatManager] Error channel full, dropping error for task %s", taskID)
				}
			}
		}
	}
}
