#!/usr/bin/env python3
"""Fail closed on mutable Helm image references.

Production values intentionally do not contain unverified image digests. They
must fail chart rendering until the release pipeline supplies a lock file with
real OCI SHA-256 digests. CI uses values-ci.yaml only to test Helm rendering;
its synthetic digests are never a deployable release lock file.
"""
from pathlib import Path
import re
import subprocess
import sys

import yaml

ROOT = Path(__file__).resolve().parents[2]
CHART = ROOT / "deploy" / "helm" / "autorix"
CI_VALUES = CHART / "values-ci.yaml"
DIGEST = re.compile(r"^[^@\s]+@sha256:[a-f0-9]{64}$")


def command(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=ROOT, capture_output=True, text=True, check=False)


def image_values(values: object):
    if isinstance(values, dict):
        image = values.get("image")
        if isinstance(image, dict):
            yield image
        for value in values.values():
            yield from image_values(value)
    elif isinstance(values, list):
        for value in values:
            yield from image_values(value)


def main() -> int:
    production = yaml.safe_load((CHART / "values.yaml").read_text())
    if production["global"]["imageDigests"].get("required") is not True:
        raise AssertionError("production values must require image digests")
    for image in image_values(production):
        if image.get("tag") in ("latest", ""):
            raise AssertionError(f"mutable or empty image tag in production values: {image}")
        if "digest" not in image:
            raise AssertionError(f"every production image needs a digest interface: {image}")

    default_render = command("helm", "template", "production-contract", str(CHART))
    if default_render.returncode == 0:
        raise AssertionError("production values rendered without verified image digests")
    if "image digest is required" not in default_render.stderr:
        raise AssertionError("production render failed for an unexpected reason: " + default_render.stderr)

    lint = command("helm", "lint", str(CHART), "-f", str(CI_VALUES))
    if lint.returncode != 0:
        raise AssertionError("CI Helm lint failed:\n" + lint.stderr)
    rendered = command("helm", "template", "ci-contract", str(CHART), "-f", str(CI_VALUES))
    if rendered.returncode != 0:
        raise AssertionError("CI Helm render failed:\n" + rendered.stderr)
    documents = [doc for doc in yaml.safe_load_all(rendered.stdout) if doc]
    images = []
    for doc in documents:
        spec = doc.get("spec", {})
        pod = spec.get("template", {}).get("spec", {})
        for container in pod.get("containers", []):
            images.append(container["image"])
    if not images:
        raise AssertionError("rendered chart had no workload images")
    invalid = [image for image in images if not DIGEST.fullmatch(image) or ":latest" in image]
    if invalid:
        raise AssertionError("rendered workloads must use SHA-256 image digests: " + ", ".join(invalid))
    print(f"Image digest contract passed ({len(images)} immutable workload images).")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (AssertionError, KeyError, OSError, yaml.YAMLError) as error:
        print(f"image digest contract failed: {error}", file=sys.stderr)
        raise SystemExit(1)
