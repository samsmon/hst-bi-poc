#!/bin/sh
# Initialise metadata (SQLite in the superset_home volume), register the
# read-only database connection, then serve with one worker.
set -e
superset db upgrade
superset fab create-admin --username admin --firstname BI --lastname Admin \
  --email admin@example.invalid --password "$SUPERSET_ADMIN_PASSWORD" || true
superset init
superset set-database-uri -d hst_dataset \
  -u "postgresql+psycopg2://bi_reader:${BI_READER_PASSWORD}@${PG_HOST}:5432/hst_dataset"
exec gunicorn -w 1 -k gthread --threads 8 --timeout 120 -b 0.0.0.0:8088 "superset.app:create_app()"
