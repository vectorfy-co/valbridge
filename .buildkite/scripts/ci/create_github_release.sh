#!/usr/bin/env bash
set -euo pipefail

GITHUB_OWNER="vectorfy-co"
GITHUB_REPO="valbridge"
GH_RELEASE_SECRET="GITHUB_TOKEN"
GH_RELEASE_GENERATE_NOTES="true"

BRANCH="$BUILDKITE_BRANCH"
if [ -z "$BRANCH" ]; then
  BRANCH="$(git rev-parse --abbrev-ref HEAD)"
fi

DEFAULT_BRANCH="main"
if [ "$BRANCH" != "$DEFAULT_BRANCH" ]; then
  echo "Not on default branch ($DEFAULT_BRANCH); skipping GitHub release creation."
  exit 0
fi

release_version=""
release_tags_json="[]"
if command -v buildkite-agent >/dev/null 2>&1; then
  release_version="$(buildkite-agent meta-data get release_version --default '' 2>/dev/null || true)"
  release_tags_json="$(buildkite-agent meta-data get release_tags_json --default '[]' 2>/dev/null || echo '[]')"
fi

RELEASE_TAGS_JSON="$release_tags_json" RELEASE_VERSION="$release_version" release_tags="$(python - <<'PY'
import json
import os

tags = json.loads(os.environ.get("RELEASE_TAGS_JSON", "[]"))
if tags:
    for tag in tags:
        print(tag)
elif os.environ.get("RELEASE_VERSION"):
    print(f'v{os.environ["RELEASE_VERSION"]}')
PY
)"

if [ -z "$release_tags" ]; then
  echo "No release_version metadata present; skipping GitHub release creation."
  exit 0
fi

gh_token="$(printenv "$GH_RELEASE_SECRET" || true)"
if [ -z "$gh_token" ]; then
  echo "$GH_RELEASE_SECRET is required for GitHub release creation"
  exit 1
fi

api_root="https://api.github.com/repos/$GITHUB_OWNER/$GITHUB_REPO"

export RELEASE_GENERATE_NOTES="$GH_RELEASE_GENERATE_NOTES"

payload_for_tag() {
  RELEASE_TAG="$1" python - <<'PY'
import json
import os

print(
    json.dumps(
        {
            "tag_name": os.environ["RELEASE_TAG"],
            "name": os.environ["RELEASE_TAG"],
            "generate_release_notes": os.environ.get("RELEASE_GENERATE_NOTES", "true") == "true",
            "draft": False,
            "prerelease": False,
        }
    )
)
PY
}

while IFS= read -r release_tag; do
  [ -n "$release_tag" ] || continue

  status_code="$(curl -sS -o /tmp/github-release-check.json -w '%{http_code}' \
    -H "Authorization: Bearer $gh_token" \
    -H "Accept: application/vnd.github+json" \
    "$api_root/releases/tags/$release_tag")"

  if [ "$status_code" = "200" ]; then
    echo "GitHub release already exists for tag $release_tag; skipping."
    continue
  fi

  payload="$(payload_for_tag "$release_tag")"

  create_code="$(curl -sS -o /tmp/github-release-create.json -w '%{http_code}' \
    -X POST \
    -H "Authorization: Bearer $gh_token" \
    -H "Accept: application/vnd.github+json" \
    -H "Content-Type: application/json" \
    -d "$payload" \
    "$api_root/releases")"

  if [ "$create_code" -lt 200 ] || [ "$create_code" -ge 300 ]; then
    echo "Failed to create GitHub release for $release_tag (HTTP $create_code)"
    cat /tmp/github-release-create.json
    exit 1
  fi

  echo "Created GitHub release: $release_tag"
done <<< "$release_tags"
