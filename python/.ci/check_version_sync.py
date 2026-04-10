from __future__ import annotations

import re
import sys
import tomllib
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]

CHECKS = [
    (
        ROOT / "packages/adapters/pydantic/pyproject.toml",
        ROOT / "packages/adapters/pydantic/src/valbridge_pydantic/__init__.py",
    ),
]

VERSION_PATTERN = re.compile(r'^__version__\s*=\s*"([^"]+)"\s*$', re.MULTILINE)


def read_pyproject_version(path: Path) -> str:
    with path.open("rb") as handle:
        data = tomllib.load(handle)
    return data["project"]["version"]


def read_module_version(path: Path) -> str:
    match = VERSION_PATTERN.search(path.read_text())
    if match is None:
        raise ValueError(f"could not find __version__ in {path}")
    return match.group(1)


def main() -> int:
    mismatches: list[str] = []

    for pyproject_path, module_path in CHECKS:
        pyproject_version = read_pyproject_version(pyproject_path)
        module_version = read_module_version(module_path)

        if pyproject_version != module_version:
            mismatches.append(
                f"{module_path.relative_to(ROOT)} has __version__={module_version}, "
                f"but {pyproject_path.relative_to(ROOT)} declares version={pyproject_version}"
            )

    if mismatches:
        for mismatch in mismatches:
            print(mismatch, file=sys.stderr)
        return 1

    print("Python package versions are in sync.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
