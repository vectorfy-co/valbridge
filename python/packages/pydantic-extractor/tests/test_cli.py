import json
from pathlib import Path

from valbridge_pydantic_extractor.cli import main


def test_cli_can_import_model_using_python_path(tmp_path, capsys):
    module_path = tmp_path / "sample_models.py"
    module_path.write_text(
        "\n".join(
            [
                "from pydantic import BaseModel",
                "",
                "class SampleModel(BaseModel):",
                "    name: str",
            ]
        )
    )

    exit_code = main(
        [
            "sample_models:SampleModel",
            "--python-path",
            str(tmp_path),
        ]
    )

    captured = capsys.readouterr()
    payload = json.loads(captured.out)

    assert exit_code == 0
    assert payload["schema"]["type"] == "object"
    assert payload["schema"]["properties"]["name"]["type"] == "string"


def test_cli_reports_import_error_and_allows_stub_module(tmp_path, capsys):
    module_path = tmp_path / "broken_models.py"
    module_path.write_text(
        "\n".join(
            [
                "import missing_dependency",
                "from pydantic import BaseModel",
                "",
                "class BrokenModel(BaseModel):",
                "    name: str",
            ]
        )
    )

    failure_code = main(
        [
            "broken_models:BrokenModel",
            "--python-path",
            str(tmp_path),
        ]
    )
    failure_payload = json.loads(capsys.readouterr().out)

    assert failure_code == 1
    assert failure_payload["schema"] is None
    assert failure_payload["diagnostics"][0]["code"] == "pydantic_extractor.import_error"
    assert "missing_dependency" in failure_payload["diagnostics"][0]["message"]

    success_code = main(
        [
            "broken_models:BrokenModel",
            "--python-path",
            str(tmp_path),
            "--stub-module",
            "missing_dependency",
        ]
    )
    success_payload = json.loads(capsys.readouterr().out)

    assert success_code == 0
    assert success_payload["schema"]["type"] == "object"


def test_cli_imports_requirements_before_loading_target(tmp_path, capsys):
    (tmp_path / "bootstrap_module.py").write_text(
        "\n".join(
            [
                "import os",
                "os.environ['BOOTSTRAPPED_VALUE'] = 'ready'",
            ]
        )
    )
    (tmp_path / "needs_bootstrap.py").write_text(
        "\n".join(
            [
                "import os",
                "from pydantic import BaseModel",
                "",
                "class NeedsBootstrap(BaseModel):",
                "    status: str = os.environ['BOOTSTRAPPED_VALUE']",
            ]
        )
    )

    exit_code = main(
        [
            "needs_bootstrap:NeedsBootstrap",
            "--python-path",
            str(tmp_path),
            "--requirement",
            "bootstrap_module",
        ]
    )

    payload = json.loads(capsys.readouterr().out)

    assert exit_code == 0
    assert payload["schema"]["properties"]["status"]["x-valbridge"]["defaultBehavior"] == {
        "kind": "default",
        "value": "ready",
    }


def test_cli_reports_invalid_target_format(capsys):
    exit_code = main(["missing-separator"])
    payload = json.loads(capsys.readouterr().out)

    assert exit_code == 1
    assert payload["diagnostics"][0]["code"] == "pydantic_extractor.invalid_target"
