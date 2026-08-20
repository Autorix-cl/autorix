#!/bin/sh
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE autorix_nexus;
    CREATE DATABASE autorix_ego;
    CREATE DATABASE autorix_janus;
    CREATE DATABASE autorix_vulcan;
    CREATE DATABASE autorix_hermes;
    CREATE DATABASE autorix_themis;
    CREATE DATABASE autorix_argus;
    CREATE DATABASE autorix_aegis;
    GRANT ALL PRIVILEGES ON DATABASE autorix_nexus TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_ego TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_janus TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_vulcan TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_hermes TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_themis TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_argus TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_aegis TO autorix;
EOSQL

# Run migrations for each service database if directories exist
if [ -d /docker-entrypoint-initdb.d/01-nexus ]; then
  for f in /docker-entrypoint-initdb.d/01-nexus/*.up.sql; do
    [ -f "$f" ] && psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "autorix_nexus" -f "$f" || true
  done
fi

if [ -d /docker-entrypoint-initdb.d/02-ego ]; then
  for f in /docker-entrypoint-initdb.d/02-ego/*.up.sql; do
    [ -f "$f" ] && psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "autorix_ego" -f "$f" || true
  done
fi

if [ -d /docker-entrypoint-initdb.d/03-janus ]; then
  for f in /docker-entrypoint-initdb.d/03-janus/*.up.sql; do
    [ -f "$f" ] && psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "autorix_janus" -f "$f" || true
  done
fi

if [ -d /docker-entrypoint-initdb.d/04-vulcan ]; then
  for f in /docker-entrypoint-initdb.d/04-vulcan/*.up.sql; do
    [ -f "$f" ] && psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "autorix_vulcan" -f "$f" || true
  done
fi

if [ -d /docker-entrypoint-initdb.d/05-hermes ]; then
  for f in /docker-entrypoint-initdb.d/05-hermes/*.up.sql; do
    [ -f "$f" ] && psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "autorix_hermes" -f "$f" || true
  done
fi

if [ -d /docker-entrypoint-initdb.d/06-themis ]; then
  for f in /docker-entrypoint-initdb.d/06-themis/*.up.sql; do
    [ -f "$f" ] && psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "autorix_themis" -f "$f" || true
  done
fi

if [ -d /docker-entrypoint-initdb.d/07-argus ]; then
  for f in /docker-entrypoint-initdb.d/07-argus/*.up.sql; do
    [ -f "$f" ] && psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "autorix_argus" -f "$f" || true
  done
fi

if [ -d /docker-entrypoint-initdb.d/08-aegis ]; then
  for f in /docker-entrypoint-initdb.d/08-aegis/*.up.sql; do
    [ -f "$f" ] && psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "autorix_aegis" -f "$f" || true
  done
fi

