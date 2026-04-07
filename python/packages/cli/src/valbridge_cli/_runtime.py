from __future__ import annotations

import importlib.metadata
import os
import platform
import shutil
import stat
import sys
import tempfile
from urllib.parse import urlparse
import urllib.request
from pathlib import Path

OWNER = "vectorfy-co"
REPO = "valbridge"
ENV_BIN = "VALBRIDGE_CLI_BIN"
DOWNLOAD_TIMEOUT_SECONDS = 30


def _package_version() -> str:
    return importlib.metadata.version("valbridge-cli")


def _platform_info() -> tuple[str, str, str]:
    machine = platform.machine().lower()
    if machine in {"x86_64", "amd64"}:
        arch = "x64"
    elif machine in {"arm64", "aarch64"}:
        arch = "arm64"
    else:
        raise RuntimeError(f"Unsupported valbridge CLI architecture: {machine}")

    if sys_platform := sys.platform:
        if sys_platform.startswith("darwin"):
            return "darwin", arch, ""
        if sys_platform.startswith("linux"):
            return "linux", arch, ""
        if sys_platform.startswith("win"):
            return "windows", arch, ".exe"

    raise RuntimeError(f"Unsupported valbridge CLI platform: {sys.platform}")


def _release_asset(version: str) -> tuple[str, str]:
    platform_name, arch, ext = _platform_info()
    filename = f"valbridge-{platform_name}-{arch}{ext}"
    tag = f"cli-v{version}"
    return filename, f"https://github.com/{OWNER}/{REPO}/releases/download/{tag}/{filename}"


def _cache_path(version: str, filename: str) -> Path:
    return Path.home() / ".cache" / "valbridge" / "cli" / version / filename


def resolve_binary() -> str:
    override = os.getenv(ENV_BIN)
    if override:
        return override

    version = _package_version()
    filename, url = _release_asset(version)
    destination = _cache_path(version, filename)
    if not destination.exists():
        destination.parent.mkdir(parents=True, exist_ok=True)
        parsed = urlparse(url)
        if parsed.scheme != "https":
            raise RuntimeError(f"Refusing to download valbridge CLI over non-HTTPS scheme: {parsed.scheme or '<missing>'}")

        with tempfile.NamedTemporaryFile(delete=False, dir=destination.parent) as temp_file:
            temp_path = Path(temp_file.name)

        try:
            with urllib.request.urlopen(url, timeout=DOWNLOAD_TIMEOUT_SECONDS) as response:
                with temp_path.open("wb") as file_obj:
                    shutil.copyfileobj(response, file_obj)
            temp_path.replace(destination)
        except Exception:
            temp_path.unlink(missing_ok=True)
            raise

        if os.name != "nt":
            destination.chmod(destination.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)

    return str(destination)
