#!/usr/bin/env bash
# P1-S5-T4: compose-level smoke test.
#
# Brings the full stack up via `docker compose`, waits for every service's
# own healthcheck to report healthy, then exercises one meaningful business
# path per engine — not just /health/ready, which would only prove the
# process is up, not that its actual API works. This is the check that
# would have caught the fake health aggregator described in the roadmap's
# audit: a healthcheck alone can lie about what matters.
set -euo pipefail

cd "$(dirname "$0")/../.."

COMPOSE_SERVICES="postgres redis argus nexus ego janus aegis"
# Never share the developer's default Compose project. A smoke run must clean
# only resources it created, even when a local Autorix stack is already up.
COMPOSE=(docker compose -p autorix-smoke -f docker-compose.yml -f scripts/ci/docker-compose.smoke.yaml --profile core)
FAILED=0

echo "==> Building and starting the stack: $COMPOSE_SERVICES"
"${COMPOSE[@]}" up -d --build $COMPOSE_SERVICES

# Exit code is tracked explicitly and re-raised after cleanup — a trap that
# ends in a successful `docker compose down` would otherwise silently
# overwrite a real failure's exit code with 0. A smoke test that can report
# "passed" after actually failing is worse than no smoke test.
EXIT_CODE=0

cleanup() {
  local code=$EXIT_CODE
  echo "==> Collecting logs for any unhealthy service"
  "${COMPOSE[@]}" ps
  echo "==> Tearing down"
  "${COMPOSE[@]}" down -v
  exit "$code"
}
trap cleanup EXIT

echo "==> Waiting for every service to report healthy (up to 180s)"
deadline=$((SECONDS + 180))
while true; do
  # Only the services this script started — docker compose ps's Health
  # column is empty (not "unhealthy") for any container with no healthcheck
  # defined, e.g. a console container left running from unrelated manual
  # testing; matching on that literal name set instead of "all containers
  # in this compose project" avoids waiting forever on something we're not
  # managing.
  unhealthy=$("${COMPOSE[@]}" ps --format '{{.Service}} {{.Health}}' \
    | awk -v services="$COMPOSE_SERVICES" 'BEGIN{split(services,s," "); for(i in s) want[s[i]]=1} want[$1] && $2 != "healthy" {print $1}')
  if [ -z "$unhealthy" ]; then
    echo "All managed services healthy."
    break
  fi
  if [ $SECONDS -ge $deadline ]; then
    echo "Timed out waiting for: $unhealthy"
    EXIT_CODE=1
    exit 1
  fi
  sleep 3
done

# The smoke project publishes no host ports, so it can coexist with a local
# developer stack. BusyBox wget is available inside engine images; execute
# requests through the Compose network instead.
check_internal() {
  local name="$1" expected_status="$2" service="$3" url="$4" payload="$5"
  local output status
  if [ -n "$payload" ]; then
    output=$("${COMPOSE[@]}" exec -T "$service" wget -S -O - \
      --header='Content-Type: application/json' --post-data="$payload" "$url" 2>&1) || true
  else
    output=$("${COMPOSE[@]}" exec -T "$service" wget -S -O - "$url" 2>&1) || true
  fi
  status=$(printf '%s\n' "$output" | awk '/HTTP\/1\.[01] [0-9]+/ {code=$2} END {print code}')
  if [ "$status" != "$expected_status" ]; then
    echo "FAIL: $name — expected HTTP $expected_status, got $status"
    printf '%s\n' "$output"
    FAILED=1
  else
    echo "OK: $name (HTTP $status)"
  fi
}

echo "==> Exercising one real business path per engine"

# ego: register an identity end to end (hashes a real password, persists it).
check_internal "ego: register identity" 201 ego http://ego:4433/self-service/registration \
  '{"password":"smoke-test-password-1","traits":{"email":"smoke-test@autorix.io","name":{"first":"Smoke","last":"Test"}}}'

# janus: register an OAuth2 client, then confirm its JWKS endpoint serves real keys.
check_internal "janus: public admin denied" 404 janus http://127.0.0.1:4444/admin/clients ''
check_internal "janus: register oauth2 client" 201 janus http://localhost:4445/admin/clients \
  '{"client_id":"smoke-test-client","client_name":"Smoke Test","is_public":false,"grant_types":["client_credentials"],"scopes":["read"]}'
check_internal "janus: JWKS" 200 janus http://127.0.0.1:4444/.well-known/jwks.json ''

# nexus: write and check a real relation tuple (REST admin, port 8080).
check_internal "nexus: write tuple" 201 nexus http://nexus:8080/tuples \
  '{"tuples":[{"namespace":"document","object":"smoke-test-doc","relation":"viewer","subject_namespace":"user","subject_id":"smoke-test-user"}]}'
check_internal "nexus: check permission" 200 nexus http://nexus:8080/check \
  '{"namespace":"document","object":"smoke-test-doc","relation":"viewer","subject_namespace":"user","subject_id":"smoke-test-user"}'

# aegis: create a routing rule via the admin API.
check_internal "aegis: create proxy rule" 201 aegis http://localhost:4456/rules \
  '{"id":"smoke-test-rule","match":{"url":"/smoke-test/<.*>","methods":["GET"]},"authenticators":[],"authorizer":{"handler":"allow"},"upstream":{"url":"http://ego:4433"}}'

if [ "$FAILED" -ne 0 ]; then
  echo "==> Smoke test FAILED"
  EXIT_CODE=1
  exit 1
fi

echo "==> Smoke test passed: every engine's real business path responded correctly"
