#!/usr/bin/env python3
"""Static contracts for the manually gated Autorix release workflow."""
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
WORKFLOW = ROOT / ".github/workflows/release-supply-chain.yml"


def require(text: str, needle: str) -> None:
    assert needle in text, f"release workflow is missing: {needle}"


def main() -> None:
    text = WORKFLOW.read_text()
    for service in ("aegis", "argus", "ego", "hermes", "janus", "nexus", "themis", "vulcan", "console"):
        require(text, f"- service: {service}")
    for goos, goarch in (("linux", "amd64"), ("linux", "arm64"), ("darwin", "amd64"), ("darwin", "arm64"), ("windows", "amd64")):
        require(text, f"- goos: {goos}")
        require(text, f"goarch: {goarch}")
    for contract in (
        "tags:\n      - 'v*'",
        "PUBLISH_ENABLED:",
        "push: false",
        "ghcr.io/${{ github.repository_owner }}/autorix-",
        "REMOTE_IMAGE_DIGEST",
        "cosign sign --yes \"$REMOTE_IMAGE_DIGEST\"",
        "id-token: write",
        "SHA256SUMS",
        "gh release create",
        "upload-release-assets: false",
        "-X main.version=$RELEASE_TAG",
    ):
        require(text, contract)
    assert "AUTORIX_REGISTRY_PASSWORD" not in text, "release path must not depend on invented registry secrets"
    print("Release workflow static contracts passed.")


if __name__ == "__main__":
    main()
