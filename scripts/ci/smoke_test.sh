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

COMPOSE_SERVICES="postgres argus nexus ego janus vulcan hermes themis aegis"
FAILED=0

echo "==> Building and starting the stack: $COMPOSE_SERVICES"
docker compose up -d --build $COMPOSE_SERVICES

# Exit code is tracked explicitly and re-raised after cleanup — a trap that
# ends in a successful `docker compose down` would otherwise silently
# overwrite a real failure's exit code with 0. A smoke test that can report
# "passed" after actually failing is worse than no smoke test.
EXIT_CODE=0

cleanup() {
  local code=$EXIT_CODE
  echo "==> Collecting logs for any unhealthy service"
  docker compose ps
  echo "==> Tearing down"
  docker compose down -v
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
  unhealthy=$(docker compose ps --format '{{.Name}} {{.Health}}' \
    | awk -v services="$COMPOSE_SERVICES" 'BEGIN{split(services,s," "); for(i in s) want["autorix-" s[i]]=1} want[$1] && $2 != "healthy" {print $1}')
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

check() {
  local name="$1" expected_status="$2"
  shift 2
  local status
  status=$(curl -s -o /tmp/smoke_body.$$ -w '%{http_code}' "$@") || status="curl_failed"
  if [ "$status" != "$expected_status" ]; then
    echo "FAIL: $name — expected HTTP $expected_status, got $status"
    cat /tmp/smoke_body.$$ 2>/dev/null || true
    echo
    FAILED=1
  else
    echo "OK: $name (HTTP $status)"
  fi
  rm -f /tmp/smoke_body.$$
}

echo "==> Exercising one real business path per engine"

# ego: register an identity end to end (hashes a real password, persists it).
check "ego: register identity" 201 -X POST http://localhost:4433/self-service/registration \
  -H "Content-Type: application/json" \
  -d '{"password":"smoke-test-password-1","traits":{"email":"smoke-test@autorix.io","name":{"first":"Smoke","last":"Test"}}}'

# janus: register an OAuth2 client, then confirm its JWKS endpoint serves real keys.
check "janus: register oauth2 client" 201 -X POST http://localhost:4444/admin/clients \
  -H "Content-Type: application/json" \
  -d '{"client_id":"smoke-test-client","client_name":"Smoke Test","is_public":false,"grant_types":["client_credentials"],"scopes":["read"]}'
check "janus: JWKS" 200 http://localhost:4444/.well-known/jwks.json

# vulcan: mint a real API key.
check "vulcan: create key" 201 -X POST http://localhost:4466/keys \
  -H "Content-Type: application/json" \
  -d '{"name":"smoke-test-key","owner_id":"smoke-test","is_live":false,"scopes":["read"]}'

# hermes: register a SAML provider, then confirm SCIM's service provider config responds.
check "hermes: register SAML provider" 201 -X POST http://localhost:4477/admin/saml/providers \
  -H "Content-Type: application/json" \
  -d '{"id":"smoke-test-idp","name":"Smoke Test IdP","ssoUrl":"https://idp.example.com/sso"}'
check "hermes: SCIM service provider config" 200 http://localhost:4477/scim/v2/ServiceProviderConfig

# nexus: write and check a real relation tuple (REST admin, port 8080).
check "nexus: write tuple" 201 -X POST http://localhost:8080/tuples \
  -H "Content-Type: application/json" \
  -d '{"tuples":[{"namespace":"document","object":"smoke-test-doc","relation":"viewer","subject_namespace":"user","subject_id":"smoke-test-user"}]}'
check "nexus: check permission" 200 -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{"namespace":"document","object":"smoke-test-doc","relation":"viewer","subject_namespace":"user","subject_id":"smoke-test-user"}'

# themis: create and evaluate a real CEL policy.
check "themis: create policy" 201 -X POST http://localhost:4488/policies \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"smoke-test","name":"smoke-test-policy","expression":"true","priority":1,"enabled":true}'
check "themis: list policies" 200 "http://localhost:4488/policies?tenant_id=smoke-test"

# aegis: create a routing rule via the admin API.
check "aegis: create proxy rule" 201 -X POST http://localhost:4456/rules \
  -H "Content-Type: application/json" \
  -d '{"id":"smoke-test-rule","match":{"url":"/smoke-test/<.*>","methods":["GET"]},"authenticators":[],"authorizer":{"handler":"allow"},"upstream":{"url":"http://ego:4433"}}'

if [ "$FAILED" -ne 0 ]; then
  echo "==> Smoke test FAILED"
  EXIT_CODE=1
  exit 1
fi

echo "==> Smoke test passed: every engine's real business path responded correctly"
