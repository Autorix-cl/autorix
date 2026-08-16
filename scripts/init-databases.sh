#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE autorix_nexus;
    CREATE DATABASE autorix_ego;
    CREATE DATABASE autorix_janus;
    CREATE DATABASE autorix_vulcan;
    CREATE DATABASE autorix_hermes;
    GRANT ALL PRIVILEGES ON DATABASE autorix_nexus TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_ego TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_janus TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_vulcan TO autorix;
    GRANT ALL PRIVILEGES ON DATABASE autorix_hermes TO autorix;
EOSQL
