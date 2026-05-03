#!/usr/bin/env bash
set -euo pipefail

GITHUB_OWNER="vectorfy-co"
GITHUB_REPO="valbridge"
GH_RELEASE_SECRET="GITHUB_TOKEN"

sha="$BUILDKITE_COMMIT"
if [ -z "$sha" ]; then
  sha="$(git rev-parse HEAD)"
fi

gh_token="$(printenv "$GH_RELEASE_SECRET" || true)"
if [ -z "$gh_token" ]; then
  echo "$GH_RELEASE_SECRET is required for CodeQL status bridge"
  exit 1
fi

api_url="https://api.github.com/repos/$GITHUB_OWNER/$GITHUB_REPO/commits/$sha/check-runs"
max_attempts=30
sleep_seconds=20
attempt=1

while [ "$attempt" -le "$max_attempts" ]; do
  echo "Polling GitHub CodeQL check-runs for $sha (attempt $attempt/$max_attempts)"

  http_code="$(curl -sS -o /tmp/codeql-check-runs.json -w '%{http_code}' \
    -H "Authorization: Bearer $gh_token" \
    -H "Accept: application/vnd.github+json" \
    "$api_url")"

  if [ "$http_code" -lt 200 ] || [ "$http_code" -ge 300 ]; then
    echo "Unable to query GitHub check-runs (HTTP $http_code)"
    cat /tmp/codeql-check-runs.json
    exit 1
  fi

  state="$(python - <<'PY'
import json
from pathlib import Path

raw = json.loads(Path('/tmp/codeql-check-runs.json').read_text(encoding='utf-8'))
runs = raw.get('check_runs', [])
codeql_runs = [r for r in runs if 'codeql' in (r.get('name', '').lower())]

if not codeql_runs:
    print('missing')
    raise SystemExit(0)

if any(r.get('status') != 'completed' for r in codeql_runs):
    print('pending')
    raise SystemExit(0)

conclusions = {r.get('conclusion') for r in codeql_runs}
if conclusions.issubset({'success', 'neutral', 'skipped'}):
    print('success')
else:
    print('failed')
PY
)"

  if [ "$state" = "success" ]; then
    echo "CodeQL check-runs succeeded."
    exit 0
  fi

  if [ "$state" = "failed" ]; then
    echo "CodeQL check-runs reported a failure."
    cat /tmp/codeql-check-runs.json
    exit 1
  fi

  sleep "$sleep_seconds"
  attempt=$((attempt + 1))
done

echo "Timed out waiting for CodeQL check-runs to complete"
exit 1
