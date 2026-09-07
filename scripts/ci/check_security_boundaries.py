#!/usr/bin/env python3
"""Render deployment defaults and guard private administration/JWT trust wiring.

Requires Helm and PyYAML, like check_helm_probes.py. This checks deployment
contracts, not running network isolation or authentication behavior.
"""
from pathlib import Path
import subprocess

import yaml

ROOT = Path(__file__).resolve().parents[2]


def assert_public_target(service, selected, deployment):
    """Resolve Service targetPort, including named container ports, before allowing ingress."""
    ports = deployment["spec"]["template"]["spec"]["containers"][0].get("ports", [])
    named_ports = {port["name"]: port["containerPort"] for port in ports if "name" in port}
    matching = [port for port in service["spec"]["ports"] if selected in (port["port"], port.get("name"))]
    assert len(matching) == 1, "Ingress must select exactly one declared service port"
    port = matching[0]
    target = port.get("targetPort", port["port"])
    if isinstance(target, str):
        assert target in named_ports, f"Unresolved ingress target port {target}"
        target = named_ports[target]
    assert target not in (4445, 4456), "Ingress must not reach an admin listener"


def env_entry(document, name):
    """Return one declared container environment entry by name."""
    entries = document["spec"]["template"]["spec"]["containers"][0].get("env", [])
    matches = [entry for entry in entries if entry["name"] == name]
    assert len(matches) == 1, f"expected exactly one {name} environment variable"
    return matches[0]


def assert_database_secret(entry, key):
    """DATABASE_URL must come from the configured Secret, never a value literal."""
    ref = entry.get("valueFrom", {}).get("secretKeyRef")
    assert ref, "DATABASE_URL must reference a Secret"
    assert ref["name"] == "autorix-db-credentials", "DATABASE_URL must use the configured credentials Secret"
    assert ref["key"] == key, f"DATABASE_URL must use {key}"


CORE_COMPOSE_SERVICES = {
    "postgres", "redis", "argus", "nexus", "ego", "janus", "aegis", "console",
}
EXTENDED_COMPOSE_SERVICES = {"vulcan", "hermes", "themis", "prometheus", "docs"}


def assert_local_compose_contract(compose):
    """Keep local onboarding credentialed, profile-explicit, and loopback-only."""
    services = compose["services"]
    assert CORE_COMPOSE_SERVICES <= services.keys(), "core Compose services are incomplete"
    assert EXTENDED_COMPOSE_SERVICES <= services.keys(), "extended Compose services are incomplete"
    for name in CORE_COMPOSE_SERVICES:
        assert services[name].get("profiles") == ["core"], f"{name} must belong to the core profile"
    for name in EXTENDED_COMPOSE_SERVICES:
        assert services[name].get("profiles") == ["extended"], f"{name} must belong to the extended profile"
    for name, service in services.items():
        for port in service.get("ports", []):
            assert isinstance(port, str) and port.startswith("127.0.0.1:"), f"{name} port must bind only localhost"
    password = services["postgres"]["environment"]["POSTGRES_PASSWORD"]
    assert password.startswith("${POSTGRES_PASSWORD:?"), "Postgres must require a generated local password"
    for name in CORE_COMPOSE_SERVICES - {"postgres", "redis", "console"}:
        database_url = services[name]["environment"].get("DATABASE_URL")
        assert database_url and "${POSTGRES_PASSWORD}" in database_url, f"{name} must use the generated database password"


def main():
    compose = yaml.safe_load((ROOT / "docker-compose.yml").read_text())["services"]
    assert_local_compose_contract({"services": compose})
    for name, forbidden in (("janus", "4445"), ("aegis", "4456")):
        for port in compose[name].get("ports", []):
            target = str(port.get("target")) if isinstance(port, dict) else str(port).split(":")[-1].split("/")[0]
            assert target != forbidden, f"{name} administration must not publish a host port"
    janus_env = compose["janus"]["environment"]
    assert janus_env["ADMIN_HOST"] == "0.0.0.0"
    assert str(janus_env["ADMIN_PORT"]) == "4445"
    assert compose["console"]["environment"]["JANUS_ADMIN_INTERNAL_URL"] == "http://janus:4445"

    rendered = subprocess.run(
        ["helm", "template", "security-check", str(ROOT / "deploy/helm/autorix"), "-f", str(ROOT / "deploy/helm/autorix/values-ci.yaml")],
        check=True, capture_output=True, text=True,
    ).stdout
    docs = [doc for doc in yaml.safe_load_all(rendered) if doc]
    services = {doc["metadata"]["name"]: doc for doc in docs if doc["kind"] == "Service"}
    deployments = {
        doc["metadata"]["labels"]["app.kubernetes.io/component"]: doc
        for doc in docs if doc["kind"] == "Deployment"
    }
    jobs = [doc for doc in docs if doc["kind"] == "Job"]
    pdbs = {
        doc["metadata"]["labels"]["app.kubernetes.io/component"]: doc
        for doc in docs if doc["kind"] == "PodDisruptionBudget"
    }
    admin_services = [svc for name, svc in services.items() if name.endswith("-janus-admin")]
    assert len(admin_services) == 1, "Janus requires a separate admin Service"
    admin = admin_services[0]
    assert admin["spec"]["type"] == "ClusterIP"
    assert admin["spec"]["ports"][0]["port"] == 4445
    assert deployments["janus"]["spec"]["replicas"] >= 1, "Janus must have at least one replica"
    for component, deployment in deployments.items():
        pod_spec = deployment["spec"]["template"]["spec"]
        assert pod_spec["terminationGracePeriodSeconds"] == 30, f"{component} needs a bounded graceful shutdown"
        spreads = pod_spec.get("topologySpreadConstraints", [])
        assert len(spreads) == 1, f"{component} needs one topology spread constraint"
        spread = spreads[0]
        assert spread["maxSkew"] == 1
        assert spread["topologyKey"] == "kubernetes.io/hostname"
        assert spread["whenUnsatisfiable"] == "ScheduleAnyway"
        assert spread["labelSelector"]["matchLabels"]["app.kubernetes.io/component"] == component
        preferences = pod_spec.get("affinity", {}).get("podAntiAffinity", {}).get("preferredDuringSchedulingIgnoredDuringExecution", [])
        assert len(preferences) == 1, f"{component} needs preferred pod anti-affinity"
        assert preferences[0]["weight"] == 100
        assert preferences[0]["podAffinityTerm"]["topologyKey"] == "kubernetes.io/hostname"
        replicas = deployment["spec"].get("replicas", 1)
        if replicas >= 2:
            assert component in pdbs, f"{component} needs a PodDisruptionBudget when replicated"
            assert pdbs[component]["spec"].get("maxUnavailable") == 1
    for doc in docs:
        if doc["kind"] != "Ingress":
            continue
        for rule in doc["spec"]["rules"]:
            for path in rule["http"]["paths"]:
                backend = path["backend"]["service"]
                assert backend["name"] != admin["metadata"]["name"], "Ingress must not reach Janus administration"
                svc = services[backend["name"]]
                selected = backend["port"].get("number", backend["port"].get("name"))
                component = svc["metadata"]["labels"]["app.kubernetes.io/component"]
                assert_public_target(svc, selected, deployments[component])
    envs = {
        name: {entry["name"]: entry.get("value") for entry in doc["spec"]["template"]["spec"]["containers"][0].get("env", [])}
        for name, doc in deployments.items()
    }
    assert envs["janus"]["ADMIN_HOST"] == "0.0.0.0"
    assert envs["janus"]["ADMIN_PORT"] == "4445"
    assert envs["janus"].get("KEY_RELOAD_INTERVAL"), "Janus must reload persisted signing keys across replicas"
    assert envs["console"]["JANUS_ADMIN_INTERNAL_URL"] == f'http://{admin["metadata"]["name"]}:4445'
    for aegis, issuer in ((compose["aegis"]["environment"], janus_env["ISSUER_URL"]), (envs["aegis"], envs["janus"]["ISSUER_URL"])):
        assert aegis["JWT_ISSUER"] == issuer
        assert aegis["JWT_AUDIENCE"] == issuer, "Default supports current Janus M2M audience only"
        assert aegis["JWT_JWKS_URL"].endswith("/.well-known/jwks.json")
    assert envs["aegis"]["JWT_JWKS_URL"].startswith("https://")
    assert envs["aegis"]["JWT_ALLOW_INSECURE_JWKS"] == "false"
    assert compose["aegis"]["environment"]["JWT_ALLOW_INSECURE_JWKS"] == "true"

    # Aegis schema changes are explicit Helm lifecycle hooks. The long-running
    # proxy gets the same Secret-backed URL but is never permitted to migrate on
    # startup. Both run with an immutable root filesystem now that rules live in
    # PostgreSQL rather than /app/rules.
    assert_database_secret(env_entry(deployments["aegis"], "DATABASE_URL"), "aegis-database-url")
    assert deployments["aegis"]["spec"]["template"]["spec"]["containers"][0]["securityContext"]["readOnlyRootFilesystem"]
    aegis_jobs = [job for job in jobs if job["metadata"]["name"].endswith("-aegis-migrate")]
    assert len(aegis_jobs) == 1, "Aegis requires exactly one migration Job"
    migration = aegis_jobs[0]
    assert migration["metadata"]["annotations"]["helm.sh/hook"] == "pre-install,pre-upgrade"
    assert migration["spec"]["backoffLimit"] >= 1
    assert migration["spec"]["template"]["spec"]["restartPolicy"] == "Never"
    migrate_container = migration["spec"]["template"]["spec"]["containers"][0]
    assert migrate_container["args"] == ["migrate"]
    assert migrate_container["image"] == deployments["aegis"]["spec"]["template"]["spec"]["containers"][0]["image"]
    assert migrate_container["securityContext"]["readOnlyRootFilesystem"]
    assert_database_secret(env_entry(migration, "DATABASE_URL"), "aegis-database-url")
    print("Deployment security boundaries passed (Compose and rendered Helm).")


if __name__ == "__main__":
    main()
