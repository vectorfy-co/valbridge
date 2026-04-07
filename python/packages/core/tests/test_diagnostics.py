from valbridge_core import Diagnostic


def test_diagnostic_round_trips_through_dict():
    diagnostic = Diagnostic(
        severity="warning",
        code="bridge.temporal.bound",
        path="$.properties.createdAt",
        message="Rendered with helper-backed temporal bound validation.",
        source="pydantic",
        target="zod",
        suggestion="Install the local zod bridge helper when using this output.",
    )

    payload = diagnostic.to_dict()

    assert payload == {
        "severity": "warning",
        "code": "bridge.temporal.bound",
        "path": "$.properties.createdAt",
        "message": "Rendered with helper-backed temporal bound validation.",
        "source": "pydantic",
        "target": "zod",
        "suggestion": "Install the local zod bridge helper when using this output.",
    }
    assert Diagnostic.from_dict(payload) == diagnostic
