-- Migration 001: Initial Schema
-- Creates core tables for ContainerLease system

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- Tenants table (organizations)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_tenants_name ON tenants(name);

-- Containers table
CREATE TABLE containers (
    id VARCHAR(36) PRIMARY KEY,
    docker_id VARCHAR(255) UNIQUE NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    image_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    cpu_milli INTEGER NOT NULL,
    memory_mb INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expiry_at TIMESTAMP NOT NULL,
    cost DECIMAL(10, 2) DEFAULT 0.0,
    error_message TEXT,
    volume_id VARCHAR(255),
    volume_size INTEGER DEFAULT 0,
    restart_count INTEGER DEFAULT 0,
    last_failure_time TIMESTAMP,
    failure_reason TEXT,
    max_restarts INTEGER DEFAULT 3
);

CREATE INDEX idx_containers_tenant_id ON containers(tenant_id);
CREATE INDEX idx_containers_docker_id ON containers(docker_id);
CREATE INDEX idx_containers_status ON containers(status);
CREATE INDEX idx_containers_expiry_at ON containers(expiry_at);

-- Leases table
CREATE TABLE leases (
    container_id VARCHAR(36) PRIMARY KEY REFERENCES containers(id) ON DELETE CASCADE,
    lease_key VARCHAR(255) UNIQUE NOT NULL,
    expiry_time TIMESTAMP NOT NULL,
    duration_minutes INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_leases_expiry_time ON leases(expiry_time);

-- Snapshots table
CREATE TABLE snapshots (
    id VARCHAR(36) PRIMARY KEY,
    container_id VARCHAR(36) NOT NULL REFERENCES containers(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    image_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    size BIGINT DEFAULT 0,
    description TEXT
);

CREATE INDEX idx_snapshots_container_id ON snapshots(container_id);
CREATE INDEX idx_snapshots_tenant_id ON snapshots(tenant_id);

-- Billing records table
CREATE TABLE billing_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    container_id VARCHAR(36) NOT NULL REFERENCES containers(id) ON DELETE SET NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    amount DECIMAL(10, 2) NOT NULL,
    cpu_milli_hours DECIMAL(15, 4) NOT NULL,
    memory_mb_hours DECIMAL(15, 4) NOT NULL,
    duration_minutes INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    period_start TIMESTAMP,
    period_end TIMESTAMP
);

CREATE INDEX idx_billing_records_tenant_id ON billing_records(tenant_id);
CREATE INDEX idx_billing_records_container_id ON billing_records(container_id);
CREATE INDEX idx_billing_records_created_at ON billing_records(created_at);

-- Audit logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    changes JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update trigger to users table
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Apply update trigger to tenants table
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
