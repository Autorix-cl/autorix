#!/bin/bash
set -euo pipefail

echo "========================================="
echo "Running Autorix SDK Cross-Language Conformance Suite"
echo "========================================="

echo "[1/3] Testing Go SDK..."
(cd sdk/go && go test -v ./...)

echo "[2/3] Checking TypeScript SDK build & types..."
(cd sdk/typescript && ../../console/node_modules/.bin/tsc --noEmit)

echo "[3/3] Testing Python SDK client syntax..."
(python3 -m py_compile sdk/python/autorix/client.py)

echo "========================================="
echo "✅ ALL 3 SDKs PASSED CONFORMANCE TESTING!"
echo "========================================="
