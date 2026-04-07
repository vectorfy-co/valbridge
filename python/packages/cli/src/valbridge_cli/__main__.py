from __future__ import annotations

import subprocess
import sys

from ._runtime import resolve_binary


def main() -> int:
    try:
        binary = resolve_binary()
    except RuntimeError as exc:
        sys.stderr.write(f"{exc}\n")
        return 1
    completed = subprocess.run([binary, *sys.argv[1:]], check=False)
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
