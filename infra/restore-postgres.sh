#!/usr/bin/env sh
set -eu

if [ -z "${DATABASE_URL:-}" ] || [ ! -f "${1:-}" ]; then
  echo "Usage: DATABASE_URL=... $0 backup.dump" >&2
  exit 1
fi

pg_restore --clean --if-exists --no-owner --no-acl --dbname="$DATABASE_URL" "$1"
