#!/usr/bin/env bash
set -euo pipefail

DEFAULT_BRANCH="main"
PACKAGE_MANAGER="pnpm"
PACKAGE_REGISTRY="npm"
CONTAINER_REGISTRY="none"
HELM_CHART_PATH="Chart.yaml"

if ! git rev-parse --verify "origin/$DEFAULT_BRANCH" >/dev/null 2>&1; then
  git fetch origin "$DEFAULT_BRANCH" --depth=200 || true
fi

BASE_REF="origin/$DEFAULT_BRANCH"
if ! git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
  BASE_REF="HEAD~1"
fi

if [ "$(git rev-parse "$BASE_REF" 2>/dev/null || echo "")" = "$(git rev-parse HEAD 2>/dev/null || echo "")" ]; then
  BASE_REF="HEAD~1"
fi

CHANGED_FILES="$(git diff --name-only "$BASE_REF"...HEAD 2>/dev/null || true)"

export CHANGED_FILES

detection_json="$(python - <<'PY'
import json
import os
from pathlib import Path

changed_files = [line.strip() for line in os.environ.get("CHANGED_FILES", "").splitlines() if line.strip()]
package_registry = os.environ.get("PACKAGE_REGISTRY", "none")
container_registry = os.environ.get("CONTAINER_REGISTRY", "none")
helm_chart_path = os.environ.get("HELM_CHART_PATH", "")
release_components = json.loads(r'''[{"component":"ts-client","ecosystem":"typescript","manifest_path":"typescript/packages/client/package.json","package_name":"@vectorfyco/valbridge","publish_enabled":true,"tag_prefix":"ts-client","workspace_root":"typescript"},{"component":"ts-core","ecosystem":"typescript","manifest_path":"typescript/packages/core/package.json","package_name":"@vectorfyco/valbridge-core","publish_enabled":true,"tag_prefix":"ts-core","workspace_root":"typescript"},{"component":"ts-zod","ecosystem":"typescript","manifest_path":"typescript/packages/adapters/zod/package.json","package_name":"@vectorfyco/valbridge-zod","publish_enabled":true,"tag_prefix":"ts-zod","workspace_root":"typescript"},{"component":"ts-zod-extractor","ecosystem":"typescript","manifest_path":"typescript/packages/zod-extractor/package.json","package_name":"@vectorfyco/valbridge-zod-extractor","publish_enabled":true,"tag_prefix":"ts-zod-extractor","workspace_root":"typescript"},{"component":"ts-zod-bridge","ecosystem":"typescript","manifest_path":"typescript/packages/zod-bridge/package.json","package_name":"@vectorfyco/valbridge-zod-bridge","publish_enabled":true,"tag_prefix":"ts-zod-bridge","workspace_root":"typescript"},{"component":"py-client","ecosystem":"python","manifest_path":"python/packages/client/pyproject.toml","package_name":"valbridge","publish_enabled":true,"tag_prefix":"py-client","workspace_root":"python"},{"component":"py-core","ecosystem":"python","manifest_path":"python/packages/core/pyproject.toml","package_name":"valbridge-core","publish_enabled":true,"tag_prefix":"py-core","workspace_root":"python"},{"component":"py-pydantic","ecosystem":"python","manifest_path":"python/packages/adapters/pydantic/pyproject.toml","package_name":"valbridge-pydantic","publish_enabled":true,"tag_prefix":"py-pydantic","workspace_root":"python"},{"component":"py-pydantic-extractor","ecosystem":"python","manifest_path":"python/packages/pydantic-extractor/pyproject.toml","package_name":"valbridge-pydantic-extractor","publish_enabled":true,"tag_prefix":"py-pydantic-extractor","workspace_root":"python"},{"component":"py-pydantic-bridge","ecosystem":"python","manifest_path":"python/packages/pydantic-bridge/pyproject.toml","package_name":"valbridge-pydantic-bridge","publish_enabled":true,"tag_prefix":"py-pydantic-bridge","workspace_root":"python"}]''')


def normalize(path: str) -> str:
    path = (path or ".").strip().strip("/")
    return "." if path in {"", "."} else path

def manifest_changed(component: dict[str, object]) -> bool:
    manifest = normalize(str(component.get("manifest_path", ".")))
    component_dir = str(Path(manifest).parent).strip(".") or "."

    for changed in changed_files:
        normalized = normalize(changed)
        if normalized == manifest:
            return True
        if component_dir != "." and (normalized == component_dir or normalized.startswith(f"{component_dir}/")):
            return True
    return False


def workspace_config_changed(component: dict[str, object]) -> bool:
    root = normalize(str(component.get("workspace_root", ".")))
    ecosystem = str(component.get("ecosystem", ""))

    workspace_files: list[str] = []
    if ecosystem == "python":
        workspace_files = ["pyproject.toml", "uv.lock"]
    elif ecosystem == "typescript":
        workspace_files = ["package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml"]

    for changed in changed_files:
        normalized = normalize(changed)
        for rel in workspace_files:
            candidate = normalize(f"{root}/{rel}" if root != "." else rel)
            if normalized == candidate:
                return True
    return False


mixed_mode = len(release_components) > 0
changed_components: list[dict[str, object]] = []
seen = set()

if mixed_mode:
    for component in release_components:
        if manifest_changed(component) or workspace_config_changed(component):
            key = str(component.get("component"))
            if key not in seen:
                changed_components.append(component)
                seen.add(key)

should_release = False
should_publish_package = False
should_build_image = False

if changed_files:
    if mixed_mode:
        should_release = len(changed_components) > 0
        should_publish_package = any(bool(component.get("publish_enabled", True)) for component in changed_components)
    elif package_registry == "pypi":
        should_publish_package = any(
            path.endswith(".py")
            or path.endswith("pyproject.toml")
            or "/src/" in f"/{path}/"
            or "/packages/" in f"/{path}/"
            or "/services/" in f"/{path}/"
            for path in changed_files
        )
        should_release = should_publish_package
    elif package_registry == "npm":
        should_publish_package = any(
            path.endswith((".ts", ".tsx", ".js", ".jsx", ".mts", ".cts"))
            or path.endswith(("package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml"))
            or "/src/" in f"/{path}/"
            or "/packages/" in f"/{path}/"
            or "/services/" in f"/{path}/"
            for path in changed_files
        )
        should_release = should_publish_package

    if container_registry == "ghcr":
        should_build_image = any(
            path.endswith(("Dockerfile", "pyproject.toml", "package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml", "Chart.yaml"))
            or any(segment in f"/{path}/" for segment in ["/docker/", "/src/", "/service/", "/services/", "/packages/", "/app/", "/deploy/", "/charts/", "/scripts/ci/"])
            for path in changed_files
        )
        should_release = should_release or should_build_image

    if helm_chart_path and normalize(helm_chart_path) in {normalize(path) for path in changed_files}:
        should_release = True

print(json.dumps({
    "should_release": should_release,
    "should_publish_package": should_publish_package,
    "should_build_image": should_build_image,
    "changed_components": changed_components,
}))
PY
)"

should_release="$(python -c 'import json,sys; print("true" if json.loads(sys.argv[1])["should_release"] else "false")' "$detection_json")"
should_publish_package="$(python -c 'import json,sys; print("true" if json.loads(sys.argv[1])["should_publish_package"] else "false")' "$detection_json")"
should_build_image="$(python -c 'import json,sys; print("true" if json.loads(sys.argv[1])["should_build_image"] else "false")' "$detection_json")"
changed_components_json="$(python -c 'import json,sys; print(json.dumps(json.loads(sys.argv[1])["changed_components"]))' "$detection_json")"

if [ "$CONTAINER_REGISTRY" = "ghcr" ] && [ "$should_build_image" = "false" ]; then
  echo "No docker-impacting change detected."
fi

if command -v buildkite-agent >/dev/null 2>&1; then
  buildkite-agent meta-data set should_release "$should_release"
  buildkite-agent meta-data set should_publish_package "$should_publish_package"
  buildkite-agent meta-data set should_build_image "$should_build_image"
  buildkite-agent meta-data set changed_release_components_json "$changed_components_json"
fi

echo "CI_SHOULD_RELEASE=$should_release" > .ci.env
echo "CI_SHOULD_PUBLISH_PACKAGE=$should_publish_package" >> .ci.env
echo "CI_CHANGED_RELEASE_COMPONENTS_JSON=$changed_components_json" >> .ci.env
if [ "$CONTAINER_REGISTRY" = "ghcr" ]; then
  echo "CI_SHOULD_BUILD_IMAGE=$should_build_image" >> .ci.env
fi

echo "Detection summary: should_release=$should_release should_publish_package=$should_publish_package should_build_image=$should_build_image"
