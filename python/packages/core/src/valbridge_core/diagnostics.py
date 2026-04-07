"""Portable diagnostics shared by converters and adapters."""

from dataclasses import dataclass
from typing import Any, Literal

DiagnosticSeverity = Literal["error", "warning", "info"]


@dataclass(frozen=True)
class Diagnostic:
    severity: DiagnosticSeverity
    code: str
    message: str
    path: str | None = None
    source: str | None = None
    target: str | None = None
    suggestion: str | None = None

    def to_dict(self) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "severity": self.severity,
            "code": self.code,
            "message": self.message,
        }
        if self.path is not None:
            payload["path"] = self.path
        if self.source is not None:
            payload["source"] = self.source
        if self.target is not None:
            payload["target"] = self.target
        if self.suggestion is not None:
            payload["suggestion"] = self.suggestion
        return payload

    @classmethod
    def from_dict(cls, payload: dict[str, Any]) -> "Diagnostic":
        return cls(
            severity=payload["severity"],
            code=payload["code"],
            message=payload["message"],
            path=payload.get("path"),
            source=payload.get("source"),
            target=payload.get("target"),
            suggestion=payload.get("suggestion"),
        )
