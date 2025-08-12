-- VeilSupport Database Initialization Script
-- This script is run when the PostgreSQL container starts

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create application user if not exists
-- This is handled by the POSTGRES_USER environment variable in docker-compose.yml

-- Basic database is created by POSTGRES_DB environment variable

-- Grant necessary permissions (if needed for production)
-- GRANT ALL PRIVILEGES ON DATABASE veilsupport TO veilsupport;