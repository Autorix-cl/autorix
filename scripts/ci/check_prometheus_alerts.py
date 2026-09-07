#!/usr/bin/env python3
"""Parse and minimally validate the repository PrometheusRule manifest."""
from pathlib import Path
import sys

import yaml


ALERTS = Path(__file__).resolve().parents[2] / "deploy" / "monitoring" / "prometheus-alerts.yaml"
NATIVE_ALERTS = Path(__file__).resolve().parents[2] / "deploy" / "monitoring" / "prometheus-alerts.yml"


def main() -> int:
    document = yaml.safe_load(ALERTS.read_text())
    if document.get("kind") != "PrometheusRule":
        raise ValueError("manifest must be a PrometheusRule")
    groups = document.get("spec", {}).get("groups", [])
    if not groups:
        raise ValueError("PrometheusRule requires at least one rule group")
    for group in groups:
        if not group.get("name") or not group.get("rules"):
            raise ValueError("every rule group requires a name and rules")
        for rule in group["rules"]:
            if not rule.get("alert") or not rule.get("expr"):
                raise ValueError("every alert rule requires alert and expr")
    native = yaml.safe_load(NATIVE_ALERTS.read_text())
    if native.get("groups") != groups:
        raise ValueError("native Prometheus rules must match the PrometheusRule manifest")
    print(f"Prometheus alert YAML parsed ({sum(len(group['rules']) for group in groups)} rules).")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, yaml.YAMLError) as error:
        print(f"Prometheus alert validation failed: {error}", file=sys.stderr)
        raise SystemExit(1)
