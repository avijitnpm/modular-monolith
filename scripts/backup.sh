#!/bin/bash
set -euo pipefail

# PostgreSQL Backup Script
# Usage: ./scripts/backup.sh [daily|weekly|monthly]
# Env vars: POSTGRES_USER, POSTGRES_DB, POSTGRES_HOST, BACKUP_DIR, BACKUP_RETAIN_DAILY, BACKUP_RETAIN_WEEKLY, BACKUP_RETAIN_MONTHLY

BACKUP_TYPE="${1:-daily}"
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-app_db}"
BACKUP_DIR="${BACKUP_DIR:-/backups}"
BACKUP_RETAIN_DAILY="${BACKUP_RETAIN_DAILY:-7}"
BACKUP_RETAIN_WEEKLY="${BACKUP_RETAIN_WEEKLY:-4}"
BACKUP_RETAIN_MONTHLY="${BACKUP_RETAIN_MONTHLY:-6}"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
FILENAME="${POSTGRES_DB}_${BACKUP_TYPE}_${TIMESTAMP}.sql.gz"
TARGET_DIR="${BACKUP_DIR}/${BACKUP_TYPE}"

mkdir -p "${TARGET_DIR}"

echo "[backup] Starting ${BACKUP_TYPE} backup of ${POSTGRES_DB}..."

pg_dump -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
  --no-owner --no-privileges --clean --if-exists \
  | gzip > "${TARGET_DIR}/${FILENAME}"

SIZE=$(du -h "${TARGET_DIR}/${FILENAME}" | cut -f1)
echo "[backup] Created: ${TARGET_DIR}/${FILENAME} (${SIZE})"

# Retention cleanup
case "${BACKUP_TYPE}" in
  daily)   RETAIN="${BACKUP_RETAIN_DAILY}" ;;
  weekly)  RETAIN="${BACKUP_RETAIN_WEEKLY}" ;;
  monthly) RETAIN="${BACKUP_RETAIN_MONTHLY}" ;;
  *)       RETAIN=7 ;;
esac

DELETED=$(find "${TARGET_DIR}" -name "*.sql.gz" -type f | sort | head -n -"${RETAIN}" | wc -l)
find "${TARGET_DIR}" -name "*.sql.gz" -type f | sort | head -n -"${RETAIN}" | xargs -r rm -f

echo "[backup] Retention: keeping ${RETAIN}, deleted ${DELETED} old backup(s)"
echo "[backup] Done."
