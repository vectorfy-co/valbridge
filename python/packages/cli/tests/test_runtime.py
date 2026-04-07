from __future__ import annotations

from valbridge_cli import _runtime


def test_release_asset_name_is_stable(monkeypatch) -> None:
    monkeypatch.setattr(_runtime.os, "sys", type("sysmod", (), {"platform": "linux"}))
    monkeypatch.setattr(_runtime.platform, "machine", lambda: "x86_64")

    filename, url = _runtime._release_asset("0.4.0")

    assert filename == "valbridge-linux-x64"
    assert url.endswith("/cli-v0.4.0/valbridge-linux-x64")
