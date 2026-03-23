// Package scheduler provides task scheduling functionality.
package scheduler

import "errors"

var (
	// ErrTaskNotFound indicates the task was not found in the queue.
	ErrTaskNotFound = errors.New("task not found in queue")

	// ErrQueueFull indicates the queue has reached its capacity limit.
	ErrQueueFull = errors.New("queue is full")

	// ErrQuotaExceeded indicates the tenant has exceeded its resource quota.
	ErrQuotaExceeded = errors.New("tenant quota exceeded")

	// ErrGlobalLimitExceeded indicates the global concurrency limit has been exceeded.
	ErrGlobalLimitExceeded = errors.New("global concurrency limit exceeded")

	// ErrDailyLimitExceeded indicates the daily task limit has been exceeded.
	ErrDailyLimitExceeded = errors.New("daily task limit exceeded")

	// ErrPreemptionFailed indicates the preemption operation failed.
	ErrPreemptionFailed = errors.New("preemption failed")

	// ErrSchedulerNotRunning indicates the scheduler is not currently running.
	ErrSchedulerNotRunning = errors.New("scheduler is not running")

	// ErrSchedulerAlreadyRunning indicates the scheduler is already running.
	ErrSchedulerAlreadyRunning = errors.New("scheduler is already running")

	// ErrTaskNotRunning indicates the task is not in a running state.
	ErrTaskNotRunning = errors.New("task is not running")
)
