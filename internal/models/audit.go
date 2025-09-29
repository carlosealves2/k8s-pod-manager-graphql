package models

import (
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Action      string         `gorm:"not null;index" json:"action"`
	ResourceType string        `gorm:"not null;index" json:"resource_type"`
	ResourceName string        `gorm:"not null;index" json:"resource_name"`
	Namespace   string         `gorm:"index" json:"namespace"`
	User        string         `json:"user"`
	UserAgent   string         `json:"user_agent"`
	IP          string         `json:"ip"`
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	StatusCode  int            `json:"status_code"`
	RequestBody string         `gorm:"type:jsonb" json:"request_body,omitempty"`
	Response    string         `gorm:"type:jsonb" json:"response,omitempty"`
	Error       string         `json:"error,omitempty"`
	Duration    int64          `json:"duration_ms"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type PodOperation struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	PodName      string         `gorm:"not null;index" json:"pod_name"`
	Namespace    string         `gorm:"not null;index" json:"namespace"`
	Operation    string         `gorm:"not null;index" json:"operation"`
	Status       string         `gorm:"not null;index:idx_status_created" json:"status"`
	Controller   string         `json:"controller,omitempty"`
	ControllerName string       `json:"controller_name,omitempty"`
	Message      string         `json:"message,omitempty"`
	Details      string         `gorm:"type:jsonb" json:"details,omitempty"`
	ExecutedBy   string         `json:"executed_by"`
	ExecutedAt   time.Time      `gorm:"index:idx_status_created" json:"executed_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (PodOperation) TableName() string {
	return "pod_operations"
}

type DeploymentScale struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	DeploymentName  string         `gorm:"not null;index" json:"deployment_name"`
	Namespace       string         `gorm:"not null;index" json:"namespace"`
	PreviousReplicas int32         `json:"previous_replicas"`
	NewReplicas     int32          `json:"new_replicas"`
	Reason          string         `json:"reason,omitempty"`
	ExecutedBy      string         `json:"executed_by"`
	ExecutedAt      time.Time      `json:"executed_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DeploymentScale) TableName() string {
	return "deployment_scales"
}