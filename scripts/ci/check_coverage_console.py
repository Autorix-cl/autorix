#!/usr/bin/env python3
"""P1-S5-T6: coverage ratchet for the console (statements %, from vitest's
json-summary reporter). Same convention as check_coverage.py for the Go
modules: this only enforces the floor in coverage-floor.json, it never
edits it — raise the floor in the same change that raises real coverage.

Usage: scripts/ci/check_coverage_console.py
Assumes `npx vitest run --coverage` has already been run in console/ (its
json-summary lands at console/coverage/coverage-summary.json).
"""
import json
import sys
from pathlib import Path

TOLERANCE = 0.5  # percentage points

REPO_ROOT = Path(__file__).resolve().parents[2]


def main() -> int:
    floor = json.loads((REPO_ROOT / "coverage-floor.json").read_text())["console"]["statements"]

    summary_path = REPO_ROOT / "console" / "coverage" / "coverage-summary.json"
    if not summary_path.exists():
        print(f"missing {summary_path} — run `npx vitest run --coverage` in console/ first")
        return 1

    summary = json.loads(summary_path.read_text())
    actual = summary["total"]["statements"]["pct"]

    print(f"console: statement coverage {actual:.2f}% (floor {floor:.2f}%)")
    if actual < floor - TOLERANCE:
        print(f"FAIL: console coverage {actual:.2f}% dropped below its floor of {floor:.2f}%")
        return 1

    print("OK: console coverage holds the floor")
    return 0


if __name__ == "__main__":
    sys.exit(main())
