-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(100),
    "user" VARCHAR(255),
    user_agent TEXT,
    ip VARCHAR(45),
    method VARCHAR(10),
    path TEXT,
    status_code INTEGER,
    request_body JSONB,
    response JSONB,
    error TEXT,
    duration_ms BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_resource_name ON audit_logs(resource_name);
CREATE INDEX idx_audit_logs_namespace ON audit_logs(namespace);
CREATE INDEX idx_audit_logs_deleted_at ON audit_logs(deleted_at);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Create pod_operations table
CREATE TABLE IF NOT EXISTS pod_operations (
    id BIGSERIAL PRIMARY KEY,
    pod_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(100) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    controller VARCHAR(50),
    controller_name VARCHAR(255),
    message TEXT,
    details JSONB,
    executed_by VARCHAR(255),
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_pod_operations_pod_name ON pod_operations(pod_name);
CREATE INDEX idx_pod_operations_namespace ON pod_operations(namespace);
CREATE INDEX idx_pod_operations_operation ON pod_operations(operation);
CREATE INDEX idx_pod_operations_status_created ON pod_operations(status, executed_at DESC);
CREATE INDEX idx_pod_operations_deleted_at ON pod_operations(deleted_at);

-- Create deployment_scales table
CREATE TABLE IF NOT EXISTS deployment_scales (
    id BIGSERIAL PRIMARY KEY,
    deployment_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(100) NOT NULL,
    previous_replicas INTEGER,
    new_replicas INTEGER,
    reason TEXT,
    executed_by VARCHAR(255),
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_deployment_scales_deployment_name ON deployment_scales(deployment_name);
CREATE INDEX idx_deployment_scales_namespace ON deployment_scales(namespace);
CREATE INDEX idx_deployment_scales_deleted_at ON deployment_scales(deleted_at);
CREATE INDEX idx_deployment_scales_executed_at ON deployment_scales(executed_at DESC);