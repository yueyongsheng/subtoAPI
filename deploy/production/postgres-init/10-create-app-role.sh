#!/bin/sh
set -eu

: "${SUB2API_DB_USER:?SUB2API_DB_USER is required}"
: "${SUB2API_DB_PASSWORD:?SUB2API_DB_PASSWORD is required}"
: "${SUB2API_DB_NAME:?SUB2API_DB_NAME is required}"

psql --set=ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" \
  --dbname "$POSTGRES_DB" \
  --set=app_user="$SUB2API_DB_USER" \
  --set=app_password="$SUB2API_DB_PASSWORD" \
  --set=app_db="$SUB2API_DB_NAME" <<-'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'app_user', :'app_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app_user') \gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'app_db', :'app_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'app_db') \gexec
SQL

psql --set=ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" \
  --dbname "$SUB2API_DB_NAME" \
  --set=app_user="$SUB2API_DB_USER" <<-'SQL'
SELECT format('ALTER SCHEMA public OWNER TO %I', :'app_user') \gexec
SELECT format('GRANT ALL ON SCHEMA public TO %I', :'app_user') \gexec
SQL
