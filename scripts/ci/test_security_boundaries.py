"""Negative fixtures for ingress-to-administration regressions."""
import unittest
from pathlib import Path

from check_security_boundaries import assert_local_compose_contract, assert_public_target


class IngressTargetTests(unittest.TestCase):
    def fixture(self, target):
        service = {"spec": {"ports": [{"name": "http", "port": 4444, "targetPort": target}]}}
        deployment = {"spec": {"template": {"spec": {"containers": [{"ports": [
            {"name": "http", "containerPort": 4444},
            {"name": "admin", "containerPort": 4445},
        ]}]}}}}
        return service, deployment

    def test_accepts_public_named_target(self):
        service, deployment = self.fixture("http")
        assert_public_target(service, "http", deployment)

    def test_rejects_admin_targets_hidden_behind_public_service_port(self):
        for target in (4445, 4456, "admin"):
            with self.subTest(target=target):
                service, deployment = self.fixture(target)
                with self.assertRaisesRegex(AssertionError, "admin listener"):
                    assert_public_target(service, 4444, deployment)

    def test_rejects_unresolved_named_target(self):
        service, deployment = self.fixture("missing")
        with self.assertRaisesRegex(AssertionError, "Unresolved"):
            assert_public_target(service, 4444, deployment)


class ComposeSmokeIsolationTests(unittest.TestCase):
    def test_smoke_cleanup_uses_an_isolated_compose_project(self):
        script = (Path(__file__).resolve().parents[2] / "scripts/ci/smoke_test.sh").read_text()
        self.assertIn("-p autorix-smoke --profile core", script)
        self.assertIn('"${COMPOSE[@]}" down -v', script)


class ComposeSecurityTests(unittest.TestCase):
    def valid_compose(self):
        core = {"postgres", "redis", "argus", "nexus", "ego", "janus", "aegis", "console"}
        services = {name: {"profiles": ["core"]} for name in core}
        services["postgres"]["environment"] = {"POSTGRES_PASSWORD": "${POSTGRES_PASSWORD:?required}"}
        services["redis"]["ports"] = ["127.0.0.1:6379:6379"]
        services["console"]["ports"] = ["127.0.0.1:3000:3000"]
        for name in core - {"postgres", "redis", "console"}:
            services[name]["environment"] = {"DATABASE_URL": "postgres://autorix:${POSTGRES_PASSWORD}@postgres/db"}
            services[name]["ports"] = ["127.0.0.1:1:1"]
        for name in ("vulcan", "hermes", "themis", "prometheus", "docs"):
            services[name] = {"profiles": ["extended"], "ports": ["127.0.0.1:1:1"]}
        return {"services": services}

    def test_rejects_non_loopback_published_port(self):
        compose = self.valid_compose()
        compose["services"]["janus"]["ports"] = ["4444:4444"]
        with self.assertRaisesRegex(AssertionError, "localhost"):
            assert_local_compose_contract(compose)

    def test_accepts_profiled_loopback_compose(self):
        assert_local_compose_contract(self.valid_compose())


if __name__ == "__main__":
    unittest.main()
