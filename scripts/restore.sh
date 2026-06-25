#!/bin/bash
set -euo pipefail

# PostgreSQL Restore Script
# Usage: ./scripts/restore.sh <backup_file.sql.gz>

BACKUP_FILE="${1:?Usage: restore.sh <backup_file.sql.gz>}"
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-app_db}"

if [ ! -f "${BACKUP_FILE}" ]; then
  echo "[restore] ERROR: File not found: ${BACKUP_FILE}"
  exit 1
fi

echo "[restore] WARNING: This will overwrite database '${POSTGRES_DB}' on ${POSTGRES_HOST}"
echo "[restore] Source: ${BACKUP_FILE}"
echo "[restore] Press Ctrl+C within 5 seconds to abort..."
sleep 5

echo "[restore] Restoring..."
gunzip -c "${BACKUP_FILE}" | psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" --single-transaction

echo "[restore] Done. Verify with: docker compose exec app wget -qO- http://127.0.0.1:8080/health/ready"
