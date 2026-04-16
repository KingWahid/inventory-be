-- PostgreSQL bootstrap extensions for local development.
-- This script is idempotent and safe to run multiple times.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
