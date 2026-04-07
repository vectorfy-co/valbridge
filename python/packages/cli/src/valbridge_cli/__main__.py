from __future__ import annotations

import os
import subprocess
import sys

from ._runtime import resolve_binary


def main() -> int:
    binary = resolve_binary()
    completed = subprocess.run([binary, *sys.argv[1:]], check=False)
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
