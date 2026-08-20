#!/usr/bin/env python3
"""P1-S5-T6: coverage ratchet for a single Go module.

Runs `go test -short -coverprofile=...` for the module, reads its floor
from coverage-floor.json, and fails if actual coverage drops below the
floor (with a small tolerance for measurement noise). Raising real
coverage means raising the floor in the same change — this script only
enforces the floor, it never edits it.

Usage: scripts/ci/check_coverage.py <module-dir>
"""
import json
import re
import subprocess
import sys
from pathlib import Path

TOLERANCE = 0.5  # percentage points

REPO_ROOT = Path(__file__).resolve().parents[2]


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: check_coverage.py <module-dir>", file=sys.stderr)
        return 2
    module = sys.argv[1]
    module_dir = REPO_ROOT / module

    floors = json.loads((REPO_ROOT / "coverage-floor.json").read_text())["go"]
    if module not in floors:
        print(f"no coverage floor configured for {module!r} in coverage-floor.json")
        return 1
    floor = floors[module]

    profile = module_dir / "coverage.out"
    proc = subprocess.run(
        ["go", "test", "-short", f"-coverprofile={profile.name}", "./..."],
        cwd=module_dir,
        capture_output=True,
        text=True,
    )
    print(proc.stdout)
    if proc.returncode != 0:
        print(proc.stderr, file=sys.stderr)
        print(f"go test failed for {module}")
        return 1

    func_proc = subprocess.run(
        ["go", "tool", "cover", f"-func={profile.name}"],
        cwd=module_dir,
        capture_output=True,
        text=True,
    )
    if func_proc.returncode != 0:
        print(func_proc.stderr, file=sys.stderr)
        print(f"go tool cover failed for {module}")
        return 1

    last_line = func_proc.stdout.strip().splitlines()[-1]
    match = re.search(r"([\d.]+)%", last_line)
    if not match:
        print(f"could not parse coverage total from: {last_line!r}")
        return 1
    actual = float(match.group(1))

    print(f"{module}: coverage {actual:.1f}% (floor {floor:.1f}%)")
    if actual < floor - TOLERANCE:
        print(f"FAIL: {module} coverage {actual:.1f}% dropped below its floor of {floor:.1f}%")
        return 1

    print(f"OK: {module} coverage holds the floor")
    return 0


if __name__ == "__main__":
    sys.exit(main())
