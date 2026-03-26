package executor

// MetricsRecorder defines the interface for recording task execution metrics.
type MetricsRecorder interface {
	RecordMetric(metricName string, taskID string, detail string)
}

// NoOpMetricsRecorder is a no-op metrics recorder for when metrics are disabled.
type NoOpMetricsRecorder struct{}

func (n *NoOpMetricsRecorder) RecordMetric(metricName string, taskID string, detail string) {}

// recordMetric records a metric if the metrics recorder is set.
func (e *TaskExecutor) recordMetric(metricName string, taskID string, detail string) {
	if e.metrics != nil {
		e.metrics.RecordMetric(metricName, taskID, detail)
	}
}
