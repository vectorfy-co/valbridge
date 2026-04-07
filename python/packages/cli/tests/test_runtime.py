from __future__ import annotations

import sys

import pytest

from valbridge_cli import _runtime


@pytest.mark.parametrize(
    ("platform_name", "machine", "expected_filename"),
    [
        ("linux", "x86_64", "valbridge-linux-x64"),
        ("darwin", "arm64", "valbridge-darwin-arm64"),
        ("win32", "x86_64", "valbridge-windows-x64.exe"),
    ],
)
def test_release_asset_name_is_stable(
    monkeypatch: pytest.MonkeyPatch,
    platform_name: str,
    machine: str,
    expected_filename: str,
) -> None:
    monkeypatch.setattr(sys, "platform", platform_name)
    monkeypatch.setattr(_runtime.platform, "machine", lambda: machine)

    filename, url = _runtime._release_asset("0.4.0")

    assert filename == expected_filename
    assert url.endswith(f"/cli-v0.4.0/{expected_filename}")
