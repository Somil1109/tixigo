#!/usr/bin/env sh
set -eu

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

backup_dir="${1:-./backups}"
mkdir -p "$backup_dir"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
destination="$backup_dir/tixigo-$timestamp.dump"
pg_dump --format=custom --no-owner --no-acl "$DATABASE_URL" --file="$destination"
find "$backup_dir" -type f -name 'tixigo-*.dump' -mtime +14 -delete
echo "$destination"
