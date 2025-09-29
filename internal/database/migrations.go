package database

import (
	"github.com/carlosf/k8s-pod-manager/internal/models"
)

func Migrate() error {
	return DB.AutoMigrate(
		&models.AuditLog{},
		&models.PodOperation{},
		&models.DeploymentScale{},
	)
}