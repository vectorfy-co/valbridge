#!/usr/bin/env python3
"""Buildkite pipeline generator — seeded by project-factory, owned by this repo.
Edit the CONFIG section to customize behavior. Logic below the divider is generic.
"""
import json

# ── CONFIG ────────────────────────────────────────────────────────────────────
AGENT_QUEUE = "mac-mini-m4"
IMAGE_REPO = "ghcr.io/vectorfy-co/valbridge"  # set "" to skip docker
IMAGE_PLATFORMS = "linux/arm64,linux/amd64"
DEFAULT_BRANCH = "main"
GITHUB_OWNER = "vectorfy-co"
GITHUB_REPO = "valbridge"
PACKAGE_MANAGER = "pnpm"  # uv | pnpm | none
PACKAGE_REGISTRY = "npm"  # pypi | npm | none
CONTAINER_REGISTRY = "none"  # ghcr | none
TEST_FRAMEWORK = "vitest"
CI_INFERENCE_MODE = "repo_metadata"  # explicit | repo_metadata

LEGACY_UV_WORKSPACE = False
LEGACY_UV_PACKAGES = json.loads(r'''[]''')
PYTHON_WORKSPACES = json.loads(r'''[{"name":"python","packages":[{"mypy_targets":["packages/core/src"],"name":"valbridge-core","path":"packages/core","pytest_target":"packages/core/tests","ruff_targets":[],"sync_args":"--all-packages --all-extras --frozen","workdir":"."},{"mypy_targets":["packages/client/src"],"name":"valbridge","path":"packages/client","pytest_target":"packages/client/tests","ruff_targets":[],"sync_args":"--all-packages --all-extras --frozen","workdir":"."},{"mypy_targets":["packages/adapters/pydantic/src"],"name":"valbridge-pydantic","path":"packages/adapters/pydantic","pytest_target":"packages/adapters/pydantic/tests","ruff_targets":[],"sync_args":"--all-packages --all-extras --frozen","workdir":"."},{"mypy_targets":["packages/pydantic-extractor/src"],"name":"valbridge-pydantic-extractor","path":"packages/pydantic-extractor","pytest_target":".","ruff_targets":[],"sync_args":"--all-packages --all-extras --frozen","workdir":"."},{"mypy_targets":["packages/pydantic-bridge/src"],"name":"valbridge-pydantic-bridge","path":"packages/pydantic-bridge","pytest_target":"","ruff_targets":["packages/pydantic-bridge/src"],"sync_args":"--all-packages --all-extras --frozen","workdir":"."}],"root_path":"python","uv_workspace":true}]''')
TYPESCRIPT_WORKSPACES = json.loads(r'''[{"name":"typescript","package_manager":"pnpm","publish":true,"root_path":"typescript","test_framework":"vitest"}]''')
RELEASE_COMPONENTS = json.loads(r'''[{"component":"ts-client","ecosystem":"typescript","manifest_path":"typescript/packages/client/package.json","package_name":"@vectorfyco/valbridge","publish_enabled":true,"tag_prefix":"ts-client","workspace_root":"typescript"},{"component":"ts-core","ecosystem":"typescript","manifest_path":"typescript/packages/core/package.json","package_name":"@vectorfyco/valbridge-core","publish_enabled":true,"tag_prefix":"ts-core","workspace_root":"typescript"},{"component":"ts-zod","ecosystem":"typescript","manifest_path":"typescript/packages/adapters/zod/package.json","package_name":"@vectorfyco/valbridge-zod","publish_enabled":true,"tag_prefix":"ts-zod","workspace_root":"typescript"},{"component":"ts-zod-extractor","ecosystem":"typescript","manifest_path":"typescript/packages/zod-extractor/package.json","package_name":"@vectorfyco/valbridge-zod-extractor","publish_enabled":true,"tag_prefix":"ts-zod-extractor","workspace_root":"typescript"},{"component":"ts-zod-bridge","ecosystem":"typescript","manifest_path":"typescript/packages/zod-bridge/package.json","package_name":"@vectorfyco/valbridge-zod-bridge","publish_enabled":true,"tag_prefix":"ts-zod-bridge","workspace_root":"typescript"},{"component":"py-client","ecosystem":"python","manifest_path":"python/packages/client/pyproject.toml","package_name":"valbridge","publish_enabled":true,"tag_prefix":"py-client","workspace_root":"python"},{"component":"py-core","ecosystem":"python","manifest_path":"python/packages/core/pyproject.toml","package_name":"valbridge-core","publish_enabled":true,"tag_prefix":"py-core","workspace_root":"python"},{"component":"py-pydantic","ecosystem":"python","manifest_path":"python/packages/adapters/pydantic/pyproject.toml","package_name":"valbridge-pydantic","publish_enabled":true,"tag_prefix":"py-pydantic","workspace_root":"python"},{"component":"py-pydantic-extractor","ecosystem":"python","manifest_path":"python/packages/pydantic-extractor/pyproject.toml","package_name":"valbridge-pydantic-extractor","publish_enabled":true,"tag_prefix":"py-pydantic-extractor","workspace_root":"python"},{"component":"py-pydantic-bridge","ecosystem":"python","manifest_path":"python/packages/pydantic-bridge/pyproject.toml","package_name":"valbridge-pydantic-bridge","publish_enabled":true,"tag_prefix":"py-pydantic-bridge","workspace_root":"python"}]''')
DOCKER_SERVICES = json.loads(r'''[]''')

ENABLE_TEST_SUITE = True
TEST_SUITE_TOKEN = "TEST_ANALYTICS_TOKEN_VALBRIDGE_TESTS"
PUBLISH_AUTH_MODE = "token"  # token | trusted_publishing
PYPI_SECRET_KEY = "PYPI_TOKEN"
NPM_SECRET_KEY = "NPM_TOKEN"
GH_RELEASE_SECRET = "GITHUB_TOKEN"

CHANGE_DETECTION = True
DEPLOY_SSH = False
RELEASE_AUTOMATION = True
VERSION_BUMP_POLICY = "always_patch"
VERSION_SOURCE = "auto"
HELM_CHART_PATH = "Chart.yaml"
AUTO_COMMIT_VERSION_BUMP = True
CREATE_GIT_TAG = True
CREATE_GITHUB_RELEASE = True
GH_RELEASE_GENERATE_NOTES = True
DOCKER_TAG_LATEST = True
DOCKER_TAG_MAJOR_MINOR = True
DOCKER_TAG_SHA = True
DOCKER_TAG_BRANCH = True

ENABLE_CODEQL_SCAN = True
CODEQL_BUILDKITE_BRIDGE = True
# ── END CONFIG ────────────────────────────────────────────────────────────────

import os
from pathlib import Path
import re
from buildkite_sdk import Pipeline
from buildkite_sdk.schema import CommandStep, GroupStep, WaitStep

try:
    import tomllib
except ModuleNotFoundError:  # pragma: no cover - Python 3.10 fallback
    tomllib = None

_RESULTS = ".buildkite-test-results"
_AGENTS = {"queue": AGENT_QUEUE}
MIXED_MODE = bool(RELEASE_COMPONENTS)


def _normalize_path(path: str) -> str:
    path = (path or ".").strip()
    if path in {"", "."}:
        return "."
    return str(Path(path))


def _join_repo_path(root: str, value: str) -> str:
    root = _normalize_path(root)
    value = _normalize_path(value)
    if root == ".":
        return value
    if value == ".":
        return root
    return str(Path(root) / value)


def _python_entries() -> list[dict[str, object]]:
    if MIXED_MODE:
        entries: list[dict[str, object]] = []
        for workspace in PYTHON_WORKSPACES:
            workspace_root = _normalize_path(str(workspace.get("root_path", ".")))
            workspace_mode = bool(workspace.get("uv_workspace", True))
            packages = workspace.get("packages") or []
            for package in packages:
                workdir = _join_repo_path(workspace_root, str(package.get("workdir", ".")))
                path = _normalize_path(str(package.get("path", ".")))
                pytest_target = str(package.get("pytest_target", "") or "")

                entries.append(
                    {
                        "label": str(package.get("name", "") or ""),
                        "package": str(package.get("name", "") or ""),
                        "path": path,
                        "workdir": workdir,
                        "sync_args": str(package.get("sync_args", "") or ""),
                        "ruff_targets": [_normalize_path(str(target)) for target in (package.get("ruff_targets") or [])],
                        "mypy_targets": [_normalize_path(str(target)) for target in (package.get("mypy_targets") or [])],
                        "pytest_target": pytest_target,
                        "uv_workspace": workspace_mode,
                    }
                )
        return entries

    packages = LEGACY_UV_PACKAGES or [
        {
            "name": "",
            "path": ".",
            "workdir": ".",
            "sync_args": "",
            "ruff_targets": [],
            "mypy_targets": [],
            "pytest_target": "",
        }
    ]
    return [
        {
            "label": str(package.get("name", "") or ""),
            "package": str(package.get("name", "") or ""),
            "path": _normalize_path(str(package.get("path", "."))),
            "workdir": _normalize_path(str(package.get("workdir", "."))),
            "sync_args": str(package.get("sync_args", "") or ""),
            "ruff_targets": [_normalize_path(str(target)) for target in (package.get("ruff_targets") or [])],
            "mypy_targets": [_normalize_path(str(target)) for target in (package.get("mypy_targets") or [])],
            "pytest_target": _normalize_path(str(package.get("pytest_target", "") or "")) if str(package.get("pytest_target", "") or "") else "",
            "uv_workspace": LEGACY_UV_WORKSPACE,
        }
        for package in packages
    ]


def _typescript_entries() -> list[dict[str, object]]:
    if MIXED_MODE:
        return [
            {
                "name": str(workspace.get("name", "") or ""),
                "root_path": _normalize_path(str(workspace.get("root_path", "."))),
                "package_manager": str(workspace.get("package_manager", "pnpm") or "pnpm"),
                "test_framework": str(workspace.get("test_framework", "vitest") or "vitest"),
                "publish": bool(workspace.get("publish", True)),
            }
            for workspace in TYPESCRIPT_WORKSPACES
        ]

    if PACKAGE_MANAGER != "pnpm":
        return []

    return [
        {
            "name": "root",
            "root_path": ".",
            "package_manager": "pnpm",
            "test_framework": TEST_FRAMEWORK,
            "publish": PACKAGE_REGISTRY == "npm",
        }
    ]


PYTHON_ENTRIES = _python_entries()
TYPESCRIPT_ENTRIES = _typescript_entries()
HAS_PYTHON_PUBLISH = any(
    str(component.get("ecosystem", "")) == "python" and bool(component.get("publish_enabled", True))
    for component in RELEASE_COMPONENTS
)
HAS_TYPESCRIPT_PUBLISH = any(
    str(component.get("ecosystem", "")) == "typescript" and bool(component.get("publish_enabled", True))
    for component in RELEASE_COMPONENTS
)


def _uv(cmd: str, pkg: str = "", workspace_mode: bool = False) -> str:
    prefix = f"--package {pkg} " if workspace_mode and pkg else ""
    base = f"uv run {prefix}{cmd}"
    return f'{base} || "$HOME/.local/bin/uv" run {prefix}{cmd}'


def _sync(args: str = "", workspace_mode: bool = False) -> str:
    args = args.strip() or ("--all-packages --all-extras --frozen" if workspace_mode else "--group dev --frozen")
    install = f"uv sync {args}"
    fallback = f'"$HOME/.local/bin/uv" sync {args}'
    guard = "command -v uv >/dev/null 2>&1 || curl -LsSf https://astral.sh/uv/install.sh | sh"
    return f"{guard} && {{ {install} || {fallback}; }}"


def _run_in_workdir(workdir: str, command: str) -> str:
    workdir = _normalize_path(workdir)
    if workdir != ".":
        return f'cd "{workdir}" && {command}'
    return command


def _default_ruff_targets(entry: dict[str, object]) -> list[str]:
    path = str(entry.get("path", "."))
    if bool(entry.get("uv_workspace", False)):
        return [f"{path}/src", f"{path}/tests"]
    return ["src", "tests"]


def _default_mypy_targets(entry: dict[str, object]) -> list[str]:
    path = str(entry.get("path", "."))
    if bool(entry.get("uv_workspace", False)):
        return [f"{path}/src"]
    return ["src"]


def _entry_root(entry: dict[str, object]) -> Path:
    workdir = _normalize_path(str(entry.get("workdir", ".") or "."))
    if workdir != ".":
        return Path(workdir)

    return Path(_normalize_path(str(entry.get("path", ".") or ".")))


def _load_pyproject(entry: dict[str, object]) -> dict[str, object]:
    pyproject_path = _entry_root(entry) / "pyproject.toml"
    if not pyproject_path.exists():
        return {}

    raw_text = pyproject_path.read_text(encoding="utf-8")
    if tomllib is None:
        return {"_raw_text": raw_text}

    with pyproject_path.open("rb") as f:
        data = tomllib.load(f)
    data["_raw_text"] = raw_text
    return data


def _tool_config(pyproject: dict[str, object], name: str) -> dict[str, object]:
    raw_text = pyproject.get("_raw_text")
    if isinstance(raw_text, str) and re.search(rf"(?m)^\[tool\.{re.escape(name)}(?:\.|\])", raw_text):
        return {"_present": True}

    tool = pyproject.get("tool")
    if not isinstance(tool, dict):
        return {}

    config = tool.get(name)
    return config if isinstance(config, dict) else {}


def _has_pytest_layout(entry: dict[str, object], pyproject: dict[str, object]) -> bool:
    root = _entry_root(entry)
    if (root / "tests").exists():
        return True

    return bool(_tool_config(pyproject, "pytest"))


def _should_run_ruff(entry: dict[str, object], pyproject: dict[str, object]) -> bool:
    return bool(entry.get("ruff_targets")) or bool(_tool_config(pyproject, "ruff"))


def _should_run_mypy(entry: dict[str, object], pyproject: dict[str, object]) -> bool:
    return bool(entry.get("mypy_targets")) or bool(_tool_config(pyproject, "mypy"))


def lint_step(entry: dict[str, object]) -> CommandStep | None:
    suffix = str(entry.get("label", "") or "")
    pkg = str(entry.get("package", "") or "")
    sync_args = str(entry.get("sync_args", "") or "")
    ruff_targets = [str(target) for target in (entry.get("ruff_targets") or [])]
    mypy_targets = [str(target) for target in (entry.get("mypy_targets") or [])]
    pyproject = _load_pyproject(entry)
    workspace_mode = bool(entry.get("uv_workspace", False))

    run_ruff = _should_run_ruff(entry, pyproject)
    run_mypy = _should_run_mypy(entry, pyproject)
    if CI_INFERENCE_MODE == "explicit":
        run_ruff = True
        run_mypy = True

    if not run_ruff and not run_mypy:
        return None

    if run_ruff and not ruff_targets:
        ruff_targets = _default_ruff_targets(entry)
    if run_mypy and not mypy_targets:
        mypy_targets = _default_mypy_targets(entry)

    tool_names = []
    if run_ruff:
        tool_names.append("Ruff")
    if run_mypy:
        tool_names.append("Mypy")
    label = f":mag: {' + '.join(tool_names)}{(' — ' + suffix) if suffix else ''}"
    commands = [_run_in_workdir(str(entry.get("workdir", ".")), _sync(sync_args, workspace_mode))]
    if run_ruff and ruff_targets:
        commands.append(_run_in_workdir(str(entry.get("workdir", ".")), _uv(f"ruff check {' '.join(ruff_targets)}", pkg, workspace_mode)))
    if run_mypy and mypy_targets:
        commands.append(_run_in_workdir(str(entry.get("workdir", ".")), _uv(f"mypy {' '.join(mypy_targets)}", pkg, workspace_mode)))

    return CommandStep(
        label=label,
        agents=_AGENTS,
        command=commands,
    )


def test_step(entry: dict[str, object]) -> CommandStep | None:
    suffix = str(entry.get("label", "") or "")
    pkg = str(entry.get("package", "") or "")
    path = str(entry.get("path", ".") or ".")
    sync_args = str(entry.get("sync_args", "") or "")
    pytest_target = str(entry.get("pytest_target", "") or "")
    pyproject = _load_pyproject(entry)
    workspace_mode = bool(entry.get("uv_workspace", False))

    if CI_INFERENCE_MODE == "repo_metadata" and not _has_pytest_layout(entry, pyproject):
        return None

    if not pytest_target:
        pytest_target = path if workspace_mode else "."

    label = f":test_tube: Pytest{(' — ' + suffix) if suffix else ''}"
    xml_name = (pkg or suffix or "pytest").replace("/", "-").replace(" ", "-")
    xml = f"{_RESULTS}/{xml_name}.xml"

    step = CommandStep(
        label=label,
        agents=_AGENTS,
        command=[
            'ROOT_DIR="${BUILDKITE_BUILD_CHECKOUT_PATH:-$(pwd)}"',
            _run_in_workdir(str(entry.get("workdir", ".")), _sync(sync_args, workspace_mode)),
            f'mkdir -p "$ROOT_DIR/{_RESULTS}"',
            _run_in_workdir(str(entry.get("workdir", ".")), _uv(f'pytest {pytest_target} -q --junit-xml "$ROOT_DIR/{xml}"', pkg, workspace_mode)),
        ],
    )

    if ENABLE_TEST_SUITE:
        step.secrets = [TEST_SUITE_TOKEN]
        step.plugins = [
            {
                "test-collector#v1.11.0": {
                    "files": xml,
                    "format": "junit",
                    "api-token-env-name": TEST_SUITE_TOKEN,
                }
            }
        ]

    return step


def python_verify_step() -> CommandStep | GroupStep:
    steps = []
    for entry in PYTHON_ENTRIES:
        lint = lint_step(entry)
        test = test_step(entry)
        if lint is not None:
            steps.append(lint)
        if test is not None:
            steps.append(test)

    if not steps:
        return CommandStep(
            label=":grey_question: No inferred Python checks",
            agents=_AGENTS,
            command="echo 'No Python verification steps were inferred from repository metadata; skipping.'",
        )

    return GroupStep(group=":test_tube: Verify Python", steps=steps)


def pnpm_verify_step(workspace: dict[str, object]) -> CommandStep:
    root_path = str(workspace.get("root_path", "."))
    workspace_name = str(workspace.get("name", "") or "")
    label_suffix = f" — {workspace_name}" if workspace_name and workspace_name != "root" else ""
    command = """set -euo pipefail

has_script() {
  node -e "const p=require('./package.json'); process.exit(p.scripts && p.scripts['$1'] ? 0 : 1)"
}

corepack enable >/dev/null 2>&1 || true

if [ ! -f pnpm-lock.yaml ]; then
  echo "pnpm-lock.yaml is required for PNPM-managed projects"
  exit 1
fi

pnpm install --frozen-lockfile

if [ -f pnpm-workspace.yaml ]; then
  pnpm -r --if-present lint
  if [ "$TEST_FRAMEWORK" = "jest" ]; then
    pnpm -r --if-present test -- --ci
  else
    pnpm -r --if-present test
  fi
  pnpm -r --if-present build
else
  if has_script lint; then
    pnpm lint
  fi
  if has_script test; then
    if [ "$TEST_FRAMEWORK" = "jest" ]; then
      pnpm test -- --ci
    else
      pnpm test
    fi
  fi
  if has_script build; then
    pnpm build
  fi
fi
"""
    return CommandStep(
        label=f":test_tube: Verify PNPM workspace{label_suffix}",
        agents=_AGENTS,
        env={"TEST_FRAMEWORK": str(workspace.get("test_framework", TEST_FRAMEWORK))},
        command=_run_in_workdir(root_path, command),
    )


def detect_changes_step() -> CommandStep:
    return CommandStep(
        label=":mag: Detect changes",
        key="detect_changes",
        agents=_AGENTS,
        command="bash .buildkite/scripts/ci/detect_changes.sh",
        env={
            "DEFAULT_BRANCH": DEFAULT_BRANCH,
            "PACKAGE_MANAGER": PACKAGE_MANAGER,
            "PACKAGE_REGISTRY": PACKAGE_REGISTRY,
            "CONTAINER_REGISTRY": CONTAINER_REGISTRY,
            "HELM_CHART_PATH": HELM_CHART_PATH,
        },
    )


def release_prepare_step() -> CommandStep:
    return CommandStep(
        label=":bookmark_tabs: Prepare release version",
        key="prepare_release",
        agents=_AGENTS,
        command="bash .buildkite/scripts/ci/release_prepare.sh",
        env={
            "DEFAULT_BRANCH": DEFAULT_BRANCH,
            "VERSION_BUMP_POLICY": VERSION_BUMP_POLICY,
            "VERSION_SOURCE": VERSION_SOURCE,
            "HELM_CHART_PATH": HELM_CHART_PATH,
            "AUTO_COMMIT_VERSION_BUMP": str(AUTO_COMMIT_VERSION_BUMP).lower(),
            "CREATE_GIT_TAG": str(CREATE_GIT_TAG).lower(),
        },
        step_if=f'build.branch == "{DEFAULT_BRANCH}"',
    )


def docker_step(service_name: str, dockerfile: str, image_suffix: str) -> CommandStep:
    image = IMAGE_REPO + (f"-{image_suffix}" if image_suffix else "")
    label = f":docker: Build + Push GHCR{(' — ' + service_name) if service_name else ''}"

    return CommandStep(
        label=label,
        agents=_AGENTS,
        secrets=["GHCR_USERNAME", "GHCR_TOKEN"],
        env={
            "IMAGE_NAME": image,
            "DOCKERFILE_PATH": dockerfile,
            "DEFAULT_BRANCH": DEFAULT_BRANCH,
            "CHANGE_DETECTION": str(CHANGE_DETECTION).lower(),
            "DOCKER_TAG_LATEST": str(DOCKER_TAG_LATEST).lower(),
            "DOCKER_TAG_MAJOR_MINOR": str(DOCKER_TAG_MAJOR_MINOR).lower(),
            "DOCKER_TAG_SHA": str(DOCKER_TAG_SHA).lower(),
            "DOCKER_TAG_BRANCH": str(DOCKER_TAG_BRANCH).lower(),
        },
        command="""set -euo pipefail

if [ "$CHANGE_DETECTION" = "true" ]; then
  SHOULD_BUILD="$(buildkite-agent meta-data get should_build_image --default true 2>/dev/null || echo true)"
  if [ "$SHOULD_BUILD" != "true" ]; then
    echo "No docker-impacting changes detected; skipping image build."
    exit 0
  fi
fi

if [ -z "$GHCR_USERNAME" ] || [ -z "$GHCR_TOKEN" ]; then
  echo "GHCR credentials are required"
  exit 1
fi

BRANCH="$BUILDKITE_BRANCH"
if [ -z "$BRANCH" ]; then
  BRANCH="$(git rev-parse --abbrev-ref HEAD)"
fi
BRANCH_TAG="$(printf '%s' "$BRANCH" | tr '/' '-')"
SHA_TAG="$(git rev-parse --short HEAD)"

RELEASE_VERSION="$(buildkite-agent meta-data get release_version --default '' 2>/dev/null || true)"

echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin

docker buildx create --use --name "$BUILDKITE_PIPELINE_SLUG-builder" >/dev/null 2>&1 || true

TAGS=()
if [ "$DOCKER_TAG_SHA" = "true" ]; then
  TAGS+=(--tag "$IMAGE_NAME:$SHA_TAG")
fi
if [ "$DOCKER_TAG_BRANCH" = "true" ]; then
  TAGS+=(--tag "$IMAGE_NAME:$BRANCH_TAG")
fi

if [ -n "$RELEASE_VERSION" ]; then
  TAGS+=(--tag "$IMAGE_NAME:v$RELEASE_VERSION")

  if [ "$DOCKER_TAG_MAJOR_MINOR" = "true" ]; then
    MAJOR="$(printf '%s' "$RELEASE_VERSION" | cut -d. -f1)"
    MINOR="$(printf '%s' "$RELEASE_VERSION" | cut -d. -f2)"
    TAGS+=(--tag "$IMAGE_NAME:$MAJOR")
    TAGS+=(--tag "$IMAGE_NAME:$MAJOR.$MINOR")
  fi

  if [ "$DOCKER_TAG_LATEST" = "true" ] && [ "$BRANCH" = "$DEFAULT_BRANCH" ]; then
    TAGS+=(--tag "$IMAGE_NAME:latest")
  fi
fi

VERSION_LABEL="$RELEASE_VERSION"
if [ -z "$VERSION_LABEL" ]; then
  VERSION_LABEL="$SHA_TAG"
fi

docker buildx build \
  --platform """ + IMAGE_PLATFORMS + """ \
  --file "$DOCKERFILE_PATH" \
  "${TAGS[@]}" \
  --label "org.opencontainers.image.source=https://github.com/""" + GITHUB_OWNER + "/" + GITHUB_REPO + """" \
  --label "org.opencontainers.image.revision=$BUILDKITE_COMMIT" \
  --label "org.opencontainers.image.version=$VERSION_LABEL" \
  --cache-from "type=registry,ref=$IMAGE_NAME:buildcache" \
  --cache-to "type=registry,ref=$IMAGE_NAME:buildcache,mode=max" \
  --push .
""",
    )


def legacy_pypi_publish_step() -> CommandStep:
    return CommandStep(
        label=":package: Publish to PyPI",
        agents=_AGENTS,
        secrets=[PYPI_SECRET_KEY],
        step_if=f'build.branch == "{DEFAULT_BRANCH}"',
        env={
            "PYPI_SECRET_KEY": PYPI_SECRET_KEY,
            "PUBLISH_AUTH_MODE": PUBLISH_AUTH_MODE,
        },
        command="""set -euo pipefail
should_publish_package="$(buildkite-agent meta-data get should_publish_package --default true 2>/dev/null || echo true)"
if [ "$should_publish_package" != "true" ]; then
  echo "No package-impacting changes detected; skipping PyPI publish."
  exit 0
fi

if [ "$PUBLISH_AUTH_MODE" = "trusted_publishing" ]; then
  echo "trusted_publishing mode: publish handled by GitHub Actions OIDC"
  exit 0
fi

TOKEN_VALUE="$(printenv "$PYPI_SECRET_KEY" || true)"
if [ -z "$TOKEN_VALUE" ]; then
  echo "$PYPI_SECRET_KEY is required"
  exit 1
fi

export UV_PUBLISH_TOKEN="$TOKEN_VALUE"
""" + _sync("", LEGACY_UV_WORKSPACE) + """
uv build || "$HOME/.local/bin/uv" build
uv publish --token "$UV_PUBLISH_TOKEN" dist/* || "$HOME/.local/bin/uv" publish --token "$UV_PUBLISH_TOKEN" dist/*
""",
    )


def mixed_pypi_publish_step() -> CommandStep:
    return CommandStep(
        label=":package: Publish Python packages",
        agents=_AGENTS,
        secrets=[PYPI_SECRET_KEY],
        step_if=f'build.branch == "{DEFAULT_BRANCH}"',
        env={
            "PYPI_SECRET_KEY": PYPI_SECRET_KEY,
            "PUBLISH_AUTH_MODE": PUBLISH_AUTH_MODE,
        },
        command="""set -euo pipefail
if [ "$PUBLISH_AUTH_MODE" = "trusted_publishing" ]; then
  echo "trusted_publishing mode is not supported for mixed-mode package publishing."
  exit 1
fi

TOKEN_VALUE="$(printenv "$PYPI_SECRET_KEY" || true)"
if [ -z "$TOKEN_VALUE" ]; then
  echo "$PYPI_SECRET_KEY is required"
  exit 1
fi

release_plan_json="$(buildkite-agent meta-data get release_plan_json --default '[]' 2>/dev/null || echo '[]')"
mapfile -t publish_rows < <(RELEASE_PLAN_JSON="$release_plan_json" python - <<'PY'
import json
import os

plan = json.loads(os.environ.get("RELEASE_PLAN_JSON", "[]"))
for item in plan:
    if item.get("ecosystem") == "python" and item.get("publish_enabled", True):
        print(f"{item['workspace_root']}\t{item['package_name']}")
PY
)

if [ "${#publish_rows[@]}" -eq 0 ]; then
  echo "No Python release components to publish."
  exit 0
fi

export UV_PUBLISH_TOKEN="$TOKEN_VALUE"
command -v uv >/dev/null 2>&1 || curl -LsSf https://astral.sh/uv/install.sh | sh

for row in "${publish_rows[@]}"; do
  IFS=$'\t' read -r workspace_root package_name <<< "$row"
  echo "Publishing Python package: $package_name from $workspace_root"
  (
    cd "$workspace_root"
    rm -rf dist
    uv build --package "$package_name" --no-sources || "$HOME/.local/bin/uv" build --package "$package_name" --no-sources
    uv publish --token "$UV_PUBLISH_TOKEN" dist/* || "$HOME/.local/bin/uv" publish --token "$UV_PUBLISH_TOKEN" dist/*
  )
done
""",
    )


def legacy_npm_publish_step() -> CommandStep:
    return CommandStep(
        label=":package: Publish to npm",
        agents=_AGENTS,
        secrets=[NPM_SECRET_KEY],
        step_if=f'build.branch == "{DEFAULT_BRANCH}"',
        env={
            "NPM_SECRET_KEY": NPM_SECRET_KEY,
            "PUBLISH_AUTH_MODE": PUBLISH_AUTH_MODE,
        },
        command="""set -euo pipefail
if [ ! -f pnpm-lock.yaml ]; then
  echo "pnpm-lock.yaml is required for PNPM publish"
  exit 1
fi

should_publish_package="$(buildkite-agent meta-data get should_publish_package --default true 2>/dev/null || echo true)"
if [ "$should_publish_package" != "true" ]; then
  echo "No package-impacting changes detected; skipping npm publish."
  exit 0
fi

if [ "$PUBLISH_AUTH_MODE" = "trusted_publishing" ]; then
  echo "trusted_publishing mode: publish handled by GitHub Actions OIDC"
  exit 0
fi

NPM_TOKEN_VALUE="$(printenv "$NPM_SECRET_KEY" || true)"
if [ -z "$NPM_TOKEN_VALUE" ]; then
  echo "$NPM_SECRET_KEY is required"
  exit 1
fi

corepack enable >/dev/null 2>&1 || true
printf "//registry.npmjs.org/:_authToken=%s\\n" "$NPM_TOKEN_VALUE" > "$HOME/.npmrc"

if [ -f pnpm-workspace.yaml ]; then
  pnpm -r publish --no-git-checks --access public
else
  pnpm publish --no-git-checks --access public
fi
""",
    )


def mixed_npm_publish_step() -> CommandStep:
    return CommandStep(
        label=":package: Publish TypeScript packages",
        agents=_AGENTS,
        secrets=[NPM_SECRET_KEY],
        step_if=f'build.branch == "{DEFAULT_BRANCH}"',
        env={
            "NPM_SECRET_KEY": NPM_SECRET_KEY,
            "PUBLISH_AUTH_MODE": PUBLISH_AUTH_MODE,
        },
        command="""set -euo pipefail
if [ "$PUBLISH_AUTH_MODE" = "trusted_publishing" ]; then
  echo "trusted_publishing mode is not supported for mixed-mode package publishing."
  exit 1
fi

NPM_TOKEN_VALUE="$(printenv "$NPM_SECRET_KEY" || true)"
if [ -z "$NPM_TOKEN_VALUE" ]; then
  echo "$NPM_SECRET_KEY is required"
  exit 1
fi

release_plan_json="$(buildkite-agent meta-data get release_plan_json --default '[]' 2>/dev/null || echo '[]')"
mapfile -t publish_rows < <(RELEASE_PLAN_JSON="$release_plan_json" python - <<'PY'
import json
import os

plan = json.loads(os.environ.get("RELEASE_PLAN_JSON", "[]"))
for item in plan:
    if item.get("ecosystem") == "typescript" and item.get("publish_enabled", True):
        print(f"{item['workspace_root']}\t{item['package_name']}")
PY
)

if [ "${#publish_rows[@]}" -eq 0 ]; then
  echo "No TypeScript release components to publish."
  exit 0
fi

corepack enable >/dev/null 2>&1 || true
printf "//registry.npmjs.org/:_authToken=%s\\n" "$NPM_TOKEN_VALUE" > "$HOME/.npmrc"

for row in "${publish_rows[@]}"; do
  IFS=$'\t' read -r workspace_root package_name <<< "$row"
  echo "Publishing npm package: $package_name from $workspace_root"
  (
    cd "$workspace_root"
    if [ ! -f pnpm-lock.yaml ]; then
      echo "pnpm-lock.yaml is required for PNPM publish in $workspace_root"
      exit 1
    fi
    pnpm install --frozen-lockfile
    pnpm --filter "$package_name" publish --no-git-checks --access public
  )
done
""",
    )


def github_release_step() -> CommandStep:
    return CommandStep(
        label=":github: Create GitHub release",
        agents=_AGENTS,
        secrets=[GH_RELEASE_SECRET],
        command="bash .buildkite/scripts/ci/create_github_release.sh",
        step_if=f'build.branch == "{DEFAULT_BRANCH}"',
        env={
            "GITHUB_OWNER": GITHUB_OWNER,
            "GITHUB_REPO": GITHUB_REPO,
            "GH_RELEASE_SECRET": GH_RELEASE_SECRET,
            "GH_RELEASE_GENERATE_NOTES": str(GH_RELEASE_GENERATE_NOTES).lower(),
        },
    )


def codeql_bridge_step() -> CommandStep:
    return CommandStep(
        label=":shield: Wait for GitHub CodeQL",
        agents=_AGENTS,
        secrets=[GH_RELEASE_SECRET],
        command="bash .buildkite/scripts/ci/check_codeql_status.sh",
        env={
            "GITHUB_OWNER": GITHUB_OWNER,
            "GITHUB_REPO": GITHUB_REPO,
            "GH_RELEASE_SECRET": GH_RELEASE_SECRET,
        },
    )


def deploy_step() -> CommandStep:
    return CommandStep(
        label=":rocket: Deploy via SSH",
        agents=_AGENTS,
        command="bash scripts/ci/deploy_ssh.sh",
        secrets=["DEPLOY_SSH_HOST", "DEPLOY_SSH_USER", "DEPLOY_SSH_PRIVATE_KEY", "GHCR_USERNAME", "GHCR_TOKEN"],
        step_if=f'build.branch == "{DEFAULT_BRANCH}"',
    )


def main() -> None:
    pipeline = Pipeline()
    pipeline.add_environment_variable("PROJECT_AGENT_QUEUE", AGENT_QUEUE)
    pipeline.add_environment_variable("TEST_RESULTS_DIR", _RESULTS)

    if CONTAINER_REGISTRY == "ghcr" and IMAGE_REPO:
        pipeline.add_environment_variable("IMAGE_REPO", IMAGE_REPO)
        pipeline.add_environment_variable("IMAGE_PLATFORMS", IMAGE_PLATFORMS)

    if CHANGE_DETECTION:
        pipeline.add_step(detect_changes_step())
        pipeline.add_step(WaitStep())

    if MIXED_MODE:
        if PYTHON_ENTRIES:
            pipeline.add_step(python_verify_step())
        for workspace in TYPESCRIPT_ENTRIES:
            if str(workspace.get("package_manager", "pnpm")) == "pnpm":
                pipeline.add_step(pnpm_verify_step(workspace))
    elif PACKAGE_MANAGER == "uv":
        pipeline.add_step(python_verify_step())
    elif PACKAGE_MANAGER == "pnpm":
        for workspace in TYPESCRIPT_ENTRIES:
            pipeline.add_step(pnpm_verify_step(workspace))

    if RELEASE_AUTOMATION:
        pipeline.add_step(release_prepare_step())

    if MIXED_MODE:
        if HAS_PYTHON_PUBLISH:
            pipeline.add_step(mixed_pypi_publish_step())
        if HAS_TYPESCRIPT_PUBLISH:
            pipeline.add_step(mixed_npm_publish_step())
    else:
        if PACKAGE_REGISTRY == "pypi":
            pipeline.add_step(legacy_pypi_publish_step())
        elif PACKAGE_REGISTRY == "npm":
            pipeline.add_step(legacy_npm_publish_step())

    if CONTAINER_REGISTRY == "ghcr" and IMAGE_REPO:
        if DOCKER_SERVICES:
            for service in DOCKER_SERVICES:
                pipeline.add_step(
                    docker_step(
                        str(service.get("name", "") or ""),
                        str(service.get("dockerfile", "Dockerfile") or "Dockerfile"),
                        str(service.get("image_suffix", "") or ""),
                    )
                )
        else:
            dockerfile = next((f for f in ["docker/Dockerfile", "Dockerfile"] if os.path.exists(f)), "Dockerfile")
            pipeline.add_step(docker_step("", dockerfile, ""))

    if CREATE_GITHUB_RELEASE:
        pipeline.add_step(github_release_step())

    if ENABLE_CODEQL_SCAN and CODEQL_BUILDKITE_BRIDGE:
        pipeline.add_step(codeql_bridge_step())

    if DEPLOY_SSH:
        pipeline.add_step(deploy_step())

    print(pipeline.to_yaml())


if __name__ == "__main__":
    main()
