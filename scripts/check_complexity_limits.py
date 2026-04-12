#!/usr/bin/env python3
from __future__ import annotations

import sys

from complexity_limits_analysis import analyze_targets
from complexity_limits_git import changed_source_files, diff_range


def main() -> int:
    try:
        base, head = diff_range()
        targets = changed_source_files(base, head)
    except RuntimeError as exc:
        print(f"[complexity-check] {exc}", file=sys.stderr)
        return 1

    if not targets:
        print("[complexity-check] no relevant source file changes detected.")
        return 0

    violations = analyze_targets(targets)
    if violations:
        print("[complexity-check] violations found:", file=sys.stderr)
        for item in violations:
            print(f" - {item}", file=sys.stderr)
        return 1

    print(f"[complexity-check] passed on {len(targets)} changed file(s).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
