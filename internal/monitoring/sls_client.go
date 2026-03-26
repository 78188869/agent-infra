package monitoring

import (
	"context"

	"github.com/example/agent-infra/pkg/aliyun/sls"
)

// SLSClient wraps the Aliyun SLS client for monitoring use.
type SLSClient struct {
	client *sls.Client
}

// NewSLSClient creates a new SLS monitoring client.
func NewSLSClient(cfg sls.Config) *SLSClient {
	return &SLSClient{client: sls.NewClient(cfg)}
}

// RecordEvent writes an execution log event to SLS.
func (s *SLSClient) RecordEvent(ctx context.Context, entry *sls.LogEntry) error {
	return s.client.Ingest(ctx, entry)
}
