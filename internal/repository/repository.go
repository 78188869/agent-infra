// Package repository provides data access interfaces and implementations.
package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantFilter represents filtering options for listing tenants.
type TenantFilter struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

// SetDefaults sets default values for the filter.
func (f *TenantFilter) SetDefaults() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 10
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
}

// Offset returns the calculated offset for pagination.
func (f *TenantFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// TaskFilter represents filtering options for listing tasks.
type TaskFilter struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
	TenantID string `form:"tenant_id"`
	Search   string `form:"search"`
}

// SetDefaults sets default values for the filter.
func (f *TaskFilter) SetDefaults() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 10
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
}

// Offset returns the calculated offset for pagination.
func (f *TaskFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// TemplateFilter represents filtering options for listing templates.
type TemplateFilter struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	TenantID  string `form:"tenant_id"`
	Status    string `form:"status"`
	SceneType string `form:"scene_type"`
	Search    string `form:"search"`
}

// SetDefaults sets default values for the filter.
func (f *TemplateFilter) SetDefaults() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 10
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
}

// Offset returns the calculated offset for pagination.
func (f *TemplateFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// BaseRepository defines common repository operations.
type BaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uuid.UUID) (*T, error)
	List(ctx context.Context, filter TenantFilter) ([]*T, int64, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// DB is an interface that wraps gorm.DB to allow mocking.
type DB interface {
	Create(value interface{}) *gorm.DB
	First(dest interface{}, conds ...interface{}) *gorm.DB
	Find(dest interface{}, conds ...interface{}) *gorm.DB
	Model(value interface{}) *gorm.DB
	Where(query interface{}, args ...interface{}) *gorm.DB
	Offset(offset int) *gorm.DB
	Limit(limit int) *gorm.DB
	Count(count *int64) *gorm.DB
	Updates(values interface{}) *gorm.DB
	Delete(value interface{}, conds ...interface{}) *gorm.DB
}
