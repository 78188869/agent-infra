// Package sls provides Aliyun Log Service (SLS) client utilities.
package sls

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// Config holds SLS client configuration.
type Config struct {
	Endpoint        string // e.g. "cn-hangzhou.log.aliyuncs.com"
	AccessKeyID     string
	AccessKeySecret string
	Project         string // e.g. "agent-infra-prod"
	LogStore        string // e.g. "execution-logs"
}

// Client wraps the SLS SDK client for ingestion.
type Client struct {
	cfg Config
}

// NewClient creates a new SLS client.
func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg}
}

// LogEntry represents a structured log entry per TRD §3.2 / monitoring.md §3.2.
type LogEntry struct {
	TaskID    string                 `json:"task_id"`
	TenantID  string                 `json:"tenant_id"`
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	EventName string                 `json:"event_name,omitempty"`
	Content   map[string]interface{} `json:"content,omitempty"`
	Source    string                 `json:"source"`
}

// Ingest sends a log entry to SLS.
// In MVP: logs to stdout. In production: uses SLS SDK PutLogs with batching.
func (c *Client) Ingest(ctx context.Context, entry *LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	// MVP: log to stdout so execution can be verified.
	// Production: replace with actual SLS SDK producer.Send().
	log.Printf("[SLS] project=%s logstore=%s entry=%s", c.cfg.Project, c.cfg.LogStore, string(data))
	return nil
}
