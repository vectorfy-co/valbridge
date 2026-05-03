#!/usr/bin/env bash
set -euo pipefail

DEFAULT_BRANCH="main"
PACKAGE_MANAGER="pnpm"
PACKAGE_REGISTRY="npm"
VERSION_BUMP_POLICY="always_patch"
VERSION_SOURCE="auto"
HELM_CHART_PATH="Chart.yaml"
AUTO_COMMIT_VERSION_BUMP="true"
CREATE_GIT_TAG="true"

BRANCH="$BUILDKITE_BRANCH"
if [ -z "$BRANCH" ]; then
  BRANCH="$(git rev-parse --abbrev-ref HEAD)"
fi

if [ "$BRANCH" != "$DEFAULT_BRANCH" ]; then
  echo "Not on default branch ($DEFAULT_BRANCH); skipping release prepare."
  exit 0
fi

should_release="true"
if command -v buildkite-agent >/dev/null 2>&1; then
  should_release="$(buildkite-agent meta-data get should_release --default true 2>/dev/null || echo true)"
fi

if [ "$should_release" != "true" ]; then
  echo "Change detection indicates no release needed; skipping version bump."
  exit 0
fi

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

COMMIT_MESSAGES="$(git log --format=%s "$BASE_REF"..HEAD 2>/dev/null || true)"
CHANGED_FILES="$(git diff --name-only "$BASE_REF"...HEAD 2>/dev/null || true)"

export PACKAGE_MANAGER PACKAGE_REGISTRY VERSION_BUMP_POLICY VERSION_SOURCE HELM_CHART_PATH COMMIT_MESSAGES CHANGED_FILES

release_output="$(python - <<'PY'
import json
import os
import re
import sys
from pathlib import Path

policy = os.getenv("VERSION_BUMP_POLICY", "always_patch")
source = os.getenv("VERSION_SOURCE", "auto")
package_manager = os.getenv("PACKAGE_MANAGER", "none")
package_registry = os.getenv("PACKAGE_REGISTRY", "none")
commit_messages = os.getenv("COMMIT_MESSAGES", "")
changed_files = [line.strip() for line in os.getenv("CHANGED_FILES", "").splitlines() if line.strip()]
release_components = json.loads(r'''[{"component":"ts-client","ecosystem":"typescript","manifest_path":"typescript/packages/client/package.json","package_name":"@vectorfyco/valbridge","publish_enabled":true,"tag_prefix":"ts-client","workspace_root":"typescript"},{"component":"ts-core","ecosystem":"typescript","manifest_path":"typescript/packages/core/package.json","package_name":"@vectorfyco/valbridge-core","publish_enabled":true,"tag_prefix":"ts-core","workspace_root":"typescript"},{"component":"ts-zod","ecosystem":"typescript","manifest_path":"typescript/packages/adapters/zod/package.json","package_name":"@vectorfyco/valbridge-zod","publish_enabled":true,"tag_prefix":"ts-zod","workspace_root":"typescript"},{"component":"ts-zod-extractor","ecosystem":"typescript","manifest_path":"typescript/packages/zod-extractor/package.json","package_name":"@vectorfyco/valbridge-zod-extractor","publish_enabled":true,"tag_prefix":"ts-zod-extractor","workspace_root":"typescript"},{"component":"ts-zod-bridge","ecosystem":"typescript","manifest_path":"typescript/packages/zod-bridge/package.json","package_name":"@vectorfyco/valbridge-zod-bridge","publish_enabled":true,"tag_prefix":"ts-zod-bridge","workspace_root":"typescript"},{"component":"py-client","ecosystem":"python","manifest_path":"python/packages/client/pyproject.toml","package_name":"valbridge","publish_enabled":true,"tag_prefix":"py-client","workspace_root":"python"},{"component":"py-core","ecosystem":"python","manifest_path":"python/packages/core/pyproject.toml","package_name":"valbridge-core","publish_enabled":true,"tag_prefix":"py-core","workspace_root":"python"},{"component":"py-pydantic","ecosystem":"python","manifest_path":"python/packages/adapters/pydantic/pyproject.toml","package_name":"valbridge-pydantic","publish_enabled":true,"tag_prefix":"py-pydantic","workspace_root":"python"},{"component":"py-pydantic-extractor","ecosystem":"python","manifest_path":"python/packages/pydantic-extractor/pyproject.toml","package_name":"valbridge-pydantic-extractor","publish_enabled":true,"tag_prefix":"py-pydantic-extractor","workspace_root":"python"},{"component":"py-pydantic-bridge","ecosystem":"python","manifest_path":"python/packages/pydantic-bridge/pyproject.toml","package_name":"valbridge-pydantic-bridge","publish_enabled":true,"tag_prefix":"py-pydantic-bridge","workspace_root":"python"}]''')


def normalize(path: str) -> str:
    path = (path or ".").strip().strip("/")
    return "." if path in {"", "."} else path


def bump(current: str, level: str) -> str:
    major, minor, patch = [int(x) for x in current.split(".")]
    if level == "major":
        major += 1
        minor = 0
        patch = 0
    elif level == "minor":
        minor += 1
        patch = 0
    else:
        patch += 1
    return f"{major}.{minor}.{patch}"


def decide_level() -> str:
    if policy == "always_patch":
        return "patch"

    if policy == "conventional_commits":
        lines = [line.strip() for line in commit_messages.splitlines() if line.strip()]
        if any("BREAKING CHANGE" in line or "!:" in line for line in lines):
            return "major"
        if any(line.startswith("feat") for line in lines):
            return "minor"
        return "patch"

    if policy == "path_rules":
        files = changed_files
        if any("schema" in f or "/api/" in f for f in files):
            return "minor"
        return "patch"

    return "patch"


def update_pyproject(path: Path, level: str) -> tuple[str, str]:
    content = path.read_text(encoding="utf-8")
    match = re.search(r'(?m)^version\s*=\s*"(\d+\.\d+\.\d+)"\s*$', content)
    if not match:
        raise SystemExit(f"{path} does not contain a simple version = \"x.y.z\" entry")
    current = match.group(1)
    nxt = bump(current, level)
    updated = content[:match.start(1)] + nxt + content[match.end(1):]
    path.write_text(updated, encoding="utf-8")
    return current, nxt


def update_package_json(path: Path, level: str) -> tuple[str, str]:
    data = json.loads(path.read_text(encoding="utf-8"))
    current = data.get("version")
    if not isinstance(current, str) or not re.fullmatch(r"\d+\.\d+\.\d+", current):
        raise SystemExit(f"{path} version is missing or not strict semver x.y.z")
    nxt = bump(current, level)
    data["version"] = nxt
    path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    return current, nxt


def set_package_json_version(path: Path, version: str) -> None:
    data = json.loads(path.read_text(encoding="utf-8"))
    current = data.get("version")
    if not isinstance(current, str) or not re.fullmatch(r"\d+\.\d+\.\d+", current):
        raise SystemExit(f"{path} version is missing or not strict semver x.y.z")
    data["version"] = version
    path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")


def pnpm_workspace_package_json_paths(root: Path) -> list[Path]:
    workspace_file = root / "pnpm-workspace.yaml"
    patterns: list[str] = []
    in_packages = False

    for raw_line in workspace_file.read_text(encoding="utf-8").splitlines():
        stripped = raw_line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if not in_packages:
            if stripped == "packages:":
                in_packages = True
            continue

        if raw_line == raw_line.lstrip() and not stripped.startswith("-"):
            break

        if stripped.startswith("-"):
            pattern = stripped[1:].strip().strip("'\"")
            if pattern:
                patterns.append(pattern)

    manifests: list[Path] = []
    seen: set[str] = set()
    for pattern in patterns:
        for path in sorted(root.glob(pattern)):
            package_json = path / "package.json" if path.is_dir() else path
            if package_json.name != "package.json" or not package_json.exists():
                continue
            if any(part in {"node_modules", ".git", ".terraform"} for part in package_json.parts):
                continue
            key = str(package_json.relative_to(root))
            if key in seen:
                continue
            seen.add(key)
            manifests.append(package_json)

    return manifests


def update_helm_chart(path: Path, level: str) -> tuple[str, str]:
    content = path.read_text(encoding="utf-8")
    version_pattern = re.compile(r'(?m)^(version\s*:\s*)(["\']?)(\d+\.\d+\.\d+)(["\']?)\s*$')
    app_version_pattern = re.compile(r'(?m)^(appVersion\s*:\s*)(["\']?)(\d+\.\d+\.\d+)(["\']?)\s*$')

    version_match = version_pattern.search(content)
    if not version_match:
        raise SystemExit(f"{path} does not contain a semver Chart version")

    current = version_match.group(3)
    nxt = bump(current, level)

    updated = version_pattern.sub(
        lambda m: f"{m.group(1)}{m.group(2)}{nxt}{m.group(4)}",
        content,
        count=1,
    )

    if app_version_pattern.search(updated):
        updated = app_version_pattern.sub(
            lambda m: f"{m.group(1)}{m.group(2)}{nxt}{m.group(4)}",
            updated,
            count=1,
        )

    path.write_text(updated, encoding="utf-8")
    return current, nxt


def component_changed(component: dict[str, object]) -> bool:
    manifest = normalize(str(component.get("manifest_path", ".")))
    component_dir = str(Path(manifest).parent).strip(".") or "."
    workspace_root = normalize(str(component.get("workspace_root", ".")))
    ecosystem = str(component.get("ecosystem", ""))

    workspace_files = []
    if ecosystem == "python":
        workspace_files = ["pyproject.toml", "uv.lock"]
    elif ecosystem == "typescript":
        workspace_files = ["package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml"]

    for changed in changed_files:
        normalized = normalize(changed)
        if normalized == manifest:
            return True
        if component_dir != "." and (normalized == component_dir or normalized.startswith(f"{component_dir}/")):
            return True
        for rel in workspace_files:
            candidate = normalize(f"{workspace_root}/{rel}" if workspace_root != "." else rel)
            if normalized == candidate:
                return True
    return False


root = Path(".")
level = decide_level()

if release_components:
    changed_components = [component for component in release_components if component_changed(component)]
    if not changed_components:
        print(json.dumps({
            "mode": "mixed",
            "plan": [],
            "manifests": [],
            "tags": [],
        }))
        sys.exit(0)

    plan = []
    manifests = []
    tags = []

    for component in changed_components:
        manifest_path = root / str(component["manifest_path"])
        ecosystem = str(component["ecosystem"])
        if ecosystem == "python":
            current, nxt = update_pyproject(manifest_path, level)
        elif ecosystem == "typescript":
            current, nxt = update_package_json(manifest_path, level)
        else:
            raise SystemExit(f"Unsupported release component ecosystem: {ecosystem}")

        plan_item = {
            "component": component["component"],
            "ecosystem": ecosystem,
            "workspace_root": component["workspace_root"],
            "manifest_path": component["manifest_path"],
            "package_name": component["package_name"],
            "tag_prefix": component["tag_prefix"],
            "publish_enabled": bool(component.get("publish_enabled", True)),
            "current_version": current,
            "next_version": nxt,
            "tag": f'{component["tag_prefix"]}-v{nxt}',
        }
        plan.append(plan_item)
        manifests.append(str(component["manifest_path"]))
        tags.append(plan_item["tag"])

    print(json.dumps({
        "mode": "mixed",
        "plan": plan,
        "manifests": manifests,
        "tags": tags,
    }))
    sys.exit(0)

if source == "auto":
    if (root / "pyproject.toml").exists():
        source = "pyproject"
    elif (root / "package.json").exists():
        source = "package_json"
    elif (root / os.getenv("HELM_CHART_PATH", "Chart.yaml")).exists():
        source = "helm_chart"
    else:
        raise SystemExit("Unable to auto-detect version source. Expected pyproject.toml, package.json, or Helm Chart.yaml")

if source == "pyproject":
    current, nxt = update_pyproject(root / "pyproject.toml", level)
    manifests = ["pyproject.toml"]
elif source == "package_json":
    current, nxt = update_package_json(root / "package.json", level)
    manifests = ["package.json"]
elif source == "helm_chart":
    chart_path = root / os.getenv("HELM_CHART_PATH", "Chart.yaml")
    current, nxt = update_helm_chart(chart_path, level)
    manifests = [str(chart_path)]
    if package_registry == "npm" and package_manager == "pnpm":
        if (root / "pnpm-workspace.yaml").exists():
            for package_json in pnpm_workspace_package_json_paths(root):
                set_package_json_version(package_json, nxt)
                manifests.append(str(package_json.relative_to(root)))
        elif (root / "package.json").exists():
            set_package_json_version(root / "package.json", nxt)
            manifests.append("package.json")
else:
    raise SystemExit(f"Unsupported version source: {source}")

print(json.dumps({
    "mode": "legacy",
    "manifests": manifests,
    "current": current,
    "next": nxt,
    "tag": f"v{nxt}",
}))
PY
)"

mode="$(python -c 'import json,sys; print(json.loads(sys.argv[1])["mode"])' "$release_output")"

if [ "$mode" = "mixed" ]; then
  release_plan_json="$(python -c 'import json,sys; print(json.dumps(json.loads(sys.argv[1])["plan"]))' "$release_output")"
  release_tags_json="$(python -c 'import json,sys; print(json.dumps(json.loads(sys.argv[1])["tags"]))' "$release_output")"
  mapfile -t manifest_files < <(python -c 'import json, sys; [print(item) for item in json.loads(sys.argv[1])["manifests"]]' "$release_output")
  release_count="$(python -c 'import json,sys; print(len(json.loads(sys.argv[1])["plan"]))' "$release_output")"

  if command -v buildkite-agent >/dev/null 2>&1; then
    buildkite-agent meta-data set release_plan_json "$release_plan_json"
    buildkite-agent meta-data set release_tags_json "$release_tags_json"
  fi

  if [ "$release_count" = "0" ]; then
    echo "No changed release components detected; skipping."
    exit 0
  fi

  echo "Prepared mixed release plan for $release_count components."
else
  mapfile -t manifest_files < <(python -c 'import json, sys; [print(item) for item in json.loads(sys.argv[1])["manifests"]]' "$release_output")
  old_version="$(python -c 'import json, sys; print(json.loads(sys.argv[1])["current"])' "$release_output")"
  new_version="$(python -c 'import json, sys; print(json.loads(sys.argv[1])["next"])' "$release_output")"
  release_tag="$(python -c 'import json, sys; print(json.loads(sys.argv[1])["tag"])' "$release_output")"

  echo "Version bump: $old_version -> $new_version (${manifest_files[0]})"

  if command -v buildkite-agent >/dev/null 2>&1; then
    buildkite-agent meta-data set release_version "$new_version"
    buildkite-agent meta-data set release_tag "$release_tag"
  fi
fi

if [ "$AUTO_COMMIT_VERSION_BUMP" != "true" ]; then
  echo "auto_commit_version_bump=false; skipping commit/tag push."
  exit 0
fi

git config user.email "buildkite-bot@local"
git config user.name "Buildkite Release Bot"

git add "${manifest_files[@]}"
if git diff --cached --quiet; then
  echo "No version file changes to commit; skipping."
  exit 0
fi

if [ "$mode" = "mixed" ]; then
  git commit -m "chore(release): bump mixed release components [skip ci]"
else
  git commit -m "chore(release): bump version to $release_tag [skip ci]"
fi

if [ "$CREATE_GIT_TAG" = "true" ]; then
  if [ "$mode" = "mixed" ]; then
    mapfile -t release_tags < <(python -c 'import json, sys; [print(item) for item in json.loads(sys.argv[1])] ' "$(python -c 'import json,sys; print(json.dumps(json.loads(sys.argv[1])["tags"]))' "$release_output")")
    for tag in "${release_tags[@]}"; do
      git tag -a "$tag" -m "Release $tag"
    done
  else
    git tag -a "$release_tag" -m "Release $release_tag"
  fi
fi

git push origin "HEAD:$DEFAULT_BRANCH"
if [ "$CREATE_GIT_TAG" = "true" ]; then
  if [ "$mode" = "mixed" ]; then
    for tag in "${release_tags[@]}"; do
      git push origin "$tag"
    done
  else
    git push origin "$release_tag"
  fi
fi

if [ "$mode" = "mixed" ]; then
  echo "Release prepare complete for mixed-mode components."
else
  echo "Release prepare complete: $release_tag"
fi
