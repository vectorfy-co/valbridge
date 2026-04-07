from __future__ import annotations

import argparse
import importlib
import json
import os
import sys
import types

from pydantic import BaseModel

from valbridge_core import Diagnostic

from .extract import extract_model


class ExtractorCLIError(Exception):
    def __init__(self, code: str, message: str, suggestion: str | None = None) -> None:
        super().__init__(message)
        self.code = code
        self.suggestion = suggestion


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="valbridge-pydantic-extractor")
    parser.add_argument("target", help="Model target in module:Class format")
    parser.add_argument("--module-root", action="append", default=[])
    parser.add_argument("--python-path", action="append", default=[])
    parser.add_argument("--requirement", action="append", default=[])
    parser.add_argument("--env", action="append", default=[])
    parser.add_argument("--stub-module", action="append", default=[])
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)

    try:
        _apply_env(args.env)
        _apply_paths([*args.module_root, *args.python_path])
        for stub_module in args.stub_module:
            _install_stub_module(stub_module)
        _apply_requirements(args.requirement)

        model = _load_model(args.target)
        extracted = extract_model(model)
        print(
            json.dumps(
                {
                    "schema": extracted.schema,
                    "diagnostics": [d.to_dict() for d in extracted.diagnostics],
                }
            )
        )
        return 0
    except ExtractorCLIError as exc:
        diagnostic = Diagnostic(
            severity="error",
            code=exc.code,
            message=str(exc),
            source="pydantic",
            suggestion=exc.suggestion,
        )
        print(json.dumps({"schema": None, "diagnostics": [diagnostic.to_dict()]}))
        return 1
    except Exception as exc:
        diagnostic = Diagnostic(
            severity="error",
            code="pydantic_extractor.import_error",
            message=str(exc),
            source="pydantic",
            suggestion="Adjust --python-path, --module-root, or --stub-module and retry.",
        )
        print(json.dumps({"schema": None, "diagnostics": [diagnostic.to_dict()]}))
        return 1


def _apply_env(entries: list[str]) -> None:
    for entry in entries:
        if "=" not in entry:
            raise ValueError(f"Invalid --env value '{entry}', expected KEY=VALUE")
        key, value = entry.split("=", 1)
        os.environ[key] = value


def _apply_paths(paths: list[str]) -> None:
    for path in reversed(paths):
        if path not in sys.path:
            sys.path.insert(0, path)


def _apply_requirements(requirements: list[str]) -> None:
    for requirement in requirements:
        try:
            importlib.import_module(requirement)
        except Exception as exc:  # pragma: no cover - defensive path
            raise ExtractorCLIError(
                "pydantic_extractor.bootstrap_error",
                f"Failed to import requirement '{requirement}': {exc}",
                "Adjust --requirement, --python-path, or --module-root and retry.",
            ) from exc


def _install_stub_module(module_name: str) -> None:
    parts = module_name.split(".")
    for index in range(1, len(parts) + 1):
        current = ".".join(parts[:index])
        if current in sys.modules:
            continue
        module = types.ModuleType(current)
        module.__dict__["__file__"] = f"<stub {current}>"
        if index != len(parts):
            module.__path__ = []  # type: ignore[attr-defined]
        sys.modules[current] = module


def _load_model(target: str) -> type[BaseModel]:
    if ":" not in target:
        raise ExtractorCLIError(
            "pydantic_extractor.invalid_target",
            "Target must use module:Class format",
            "Use the extractor target form module_name:ModelClass.",
        )

    module_name, export_name = target.split(":", 1)
    try:
        module = importlib.import_module(module_name)
    except Exception as exc:
        raise ExtractorCLIError(
            "pydantic_extractor.import_error",
            str(exc),
            "Adjust --python-path, --module-root, or --stub-module and retry.",
        ) from exc

    try:
        model = getattr(module, export_name)
    except AttributeError as exc:
        raise ExtractorCLIError(
            "pydantic_extractor.invalid_target",
            f"{target} does not export {export_name}",
            "Check the target export name and retry.",
        ) from exc

    if not isinstance(model, type) or not issubclass(model, BaseModel):
        raise ExtractorCLIError(
            "pydantic_extractor.invalid_target",
            f"{target} is not a Pydantic BaseModel",
            "Point the extractor at a BaseModel subclass.",
        )

    return model
