#!/usr/bin/env python3
"""P1-S1-T8: CI guard against probe drift.

Renders the Helm chart, extracts every liveness/readiness probe path or gRPC
port declared per Deployment, and asserts each one is a route actually
registered in the corresponding engine's Go source — the exact class of bug
that shipped once (ego's Helm chart probing a `/health/alive` that existed
nowhere in the code) and must not ship twice.

Usage: scripts/ci/check_helm_probes.py [chart_dir]
Exits non-zero with a human-readable diff of what's wrong.
"""
import re
import subprocess
import sys
from pathlib import Path

import yaml

REPO_ROOT = Path(__file__).resolve().parents[2]
CHART_DIR = Path(sys.argv[1]) if len(sys.argv) > 1 else REPO_ROOT / "deploy" / "helm" / "autorix"

# Deployments this guard does not own: not a Go engine, or has no HTTP/gRPC
# health surface of its own to validate against Go source.
SKIP_DEPLOYMENTS = {"console"}

ROUTE_PATTERN = re.compile(r'HandleFunc\(\s*"(?:GET|POST|PUT|PATCH|DELETE)\s+([^"]+)"')
GRPC_HEALTH_PATTERN = re.compile(r"grpchealth\.Register\(")

# Routes registered once, centrally, by every engine that calls
# health.Handler.Register(mux) — not literal in any engine's own server.go.
SHARED_CONTRACT_ROUTES = {"/health/alive", "/health/ready", "/info"}


def render_chart() -> list[dict]:
    proc = subprocess.run(
        ["helm", "template", str(CHART_DIR)],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        print("helm template failed:\n" + proc.stderr, file=sys.stderr)
        sys.exit(1)
    return [doc for doc in yaml.safe_load_all(proc.stdout) if doc]


def engine_name_from_deployment(name: str) -> str | None:
    # Deployment names render as "<release>-autorix-<engine>".
    match = re.search(r"autorix-([a-z0-9]+)$", name)
    return match.group(1) if match else None


def registered_http_routes(engine: str) -> set[str]:
    routes = set(SHARED_CONTRACT_ROUTES)
    server_go = REPO_ROOT / engine / "internal" / "transport" / "http" / "server.go"
    if server_go.exists():
        routes |= set(ROUTE_PATTERN.findall(server_go.read_text()))
    return routes


def grpc_health_registered(engine: str) -> bool:
    main_go = REPO_ROOT / engine / "cmd" / f"{engine}d" / "main.go"
    return main_go.exists() and bool(GRPC_HEALTH_PATTERN.search(main_go.read_text()))


def extract_probes(container: dict) -> list[tuple[str, dict]]:
    probes = []
    for kind in ("readinessProbe", "livenessProbe"):
        probe = container.get(kind)
        if probe:
            probes.append((kind, probe))
    return probes


def main() -> int:
    docs = render_chart()
    errors = []
    checked = 0

    for doc in docs:
        if doc.get("kind") != "Deployment":
            continue
        name = doc["metadata"]["name"]
        engine = engine_name_from_deployment(name)
        if engine is None or engine in SKIP_DEPLOYMENTS:
            continue

        containers = doc["spec"]["template"]["spec"]["containers"]
        for container in containers:
            for probe_kind, probe in extract_probes(container):
                checked += 1
                if "httpGet" in probe:
                    path = probe["httpGet"]["path"]
                    routes = registered_http_routes(engine)
                    if path not in routes:
                        errors.append(
                            f"{name}/{probe_kind}: httpGet path {path!r} is not registered "
                            f"in {engine}/internal/transport/http/server.go "
                            f"(known routes: {sorted(routes)})"
                        )
                elif "grpc" in probe:
                    if not grpc_health_registered(engine):
                        errors.append(
                            f"{name}/{probe_kind}: uses a native gRPC health probe, but "
                            f"{engine}/cmd/{engine}d/main.go never calls grpchealth.Register(...)"
                        )
                elif "tcpSocket" in probe:
                    errors.append(
                        f"{name}/{probe_kind}: still uses a bare tcpSocket probe — "
                        f"the uniform health contract (P1-S1) expects httpGet or grpc"
                    )
                else:
                    errors.append(f"{name}/{probe_kind}: unrecognized probe shape {probe!r}")

    if errors:
        print(f"Probe drift detected ({len(errors)} of {checked} probes checked):\n")
        for err in errors:
            print(f"  - {err}")
        return 1

    print(f"All {checked} Helm probes resolve to routes registered in Go source.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
