
#!/bin/bash

# --- RHOAM RESTORE CONFIGURATION ---
# Fill out these variables before running the main script.

# -- General --
export AWS_REGION="your-aws-region"

# -- Postgres (RDS) --
export RHSSO_DB_ID="<rhsso-db-instance-id>"
export RHSSO_DB_SNAPSHOT="<rhsso-db-snapshot-name>"

export USERSSO_DB_ID="<usersso-db-instance-id>"
export USERSSO_DB_SNAPSHOT="<usersso-db-snapshot-name>"

export THREESCALE_DB_ID="<threescale-db-instance-id>"
export THREESCALE_DB_SNAPSHOT="<threescale-db-snapshot-name>"

# -- Redis (ElastiCache) --
export RATELIMIT_REDIS_ID="<ratelimit-redis-id>"
export RATELIMIT_REDIS_SNAPSHOT="<ratelimit-redis-snapshot-name>"

export BACKEND_REDIS_ID="<backend-redis-id>"
export BACKEND_REDIS_SNAPSHOT="<backend-redis-snapshot-name>"

export SYSTEM_REDIS_ID="<system-redis-id>"
export SYSTEM_REDIS_SNAPSHOT="<system-redis-snapshot-name>"

# -- S3 --
export S3_TARGET_BUCKET="<target-s3-bucket-name>"
export S3_BACKUP_BUCKET="<backup-s3-bucket-name>"
