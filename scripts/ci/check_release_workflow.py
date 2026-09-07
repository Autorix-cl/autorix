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
        "validate-release:",
        "Validate stable SemVer tag and release notes",
        "git describe --exact-match --tags HEAD",
        "needs: validate-release",
        "push: false",
        "ghcr.io/${{ github.repository_owner }}/autorix-",
        "REMOTE_IMAGE_DIGEST",
        "cosign sign --yes \"$REMOTE_IMAGE_DIGEST\"",
        "cosign attest --yes --type spdxjson",
        "fail-build: true",
        "severity-cutoff: critical",
        "id-token: write",
        "SHA256SUMS",
        "gh release create",
        "upload-release-assets: false",
        "-X main.version=$RELEASE_TAG",
        "docker/setup-buildx-action@e468171a9de216ec08956ac3ada2f0791b6bd435",
        "docker/build-push-action@263435318d21b8e681c14492fe198d362a7d2c83",
        "docker/login-action@9780b0c442fbb1117ed29e0efdff1e18412f7567",
        "actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02",
        "actions/download-artifact@d3f86a106a0bac45b974a628896c90dbdf5c8093",
    ):
        require(text, contract)
    assert "AUTORIX_REGISTRY_PASSWORD" not in text, "release path must not depend on invented registry secrets"
    print("Release workflow static contracts passed.")


if __name__ == "__main__":
    main()
