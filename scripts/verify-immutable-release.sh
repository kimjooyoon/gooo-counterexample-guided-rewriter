#!/usr/bin/env bash
set -euo pipefail

repo=${IMMUTABLE_RELEASE_REPO:-kimjooyoon/gooo-counterexample-guided-rewriter}
api_version=2026-03-10

fail_closed() {
  echo "immutable release guard: $1" >&2
  exit 1
}

case "${1:-}" in
  --contract)
    grep -q '/immutable-releases' "$0" || fail_closed 'repository setting endpoint is not declared'
    grep -q 'enabled' "$0" || fail_closed 'setting enabled field is not checked'
    grep -q 'immutable' "$0" || fail_closed 'release immutable field is not checked'
    grep -q 'missing' "$0" || fail_closed 'missing fields are not fail-closed'
    echo 'immutable_release_guard_contract=CLOSED'
    ;;
  --setting)
    [[ -n "${GH_TOKEN:-}" ]] || fail_closed 'GH_TOKEN is required for the admin setting check'
    if ! payload=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repo/immutable-releases" 2>&1); then
      if [[ "${IMMUTABLE_RELEASE_SETTING_ALLOW_UNAVAILABLE:-false}" == 'true' && "$payload" == *'403'* ]]; then
        echo 'repository_immutable_releases=unavailable_to_actions_token; post_release_check_required'
        exit 0
      fi
      fail_closed 'repository immutable-releases endpoint is unavailable'
    fi
    enabled=$(jq -r 'if type == "object" and has("enabled") then (.enabled | tostring) else "missing" end' <<<"$payload") || fail_closed 'setting response is not valid JSON'
    [[ "$enabled" == "true" ]] || fail_closed "repository immutable releases setting is $enabled"
    echo 'repository_immutable_releases=true'
    ;;
  --release)
    tag=${2:-}
    [[ -n "$tag" ]] || fail_closed 'release tag is required'
    if ! payload=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repo/releases/tags/$tag"); then
      fail_closed "release $tag cannot be read"
    fi
    immutable=$(jq -r 'if type == "object" and has("immutable") then (.immutable | tostring) else "missing" end' <<<"$payload") || fail_closed 'release response is not valid JSON'
    [[ "$immutable" == "true" ]] || fail_closed "release $tag immutable=$immutable"
    echo "release=$tag immutable=true"
    ;;
  *)
    fail_closed 'usage: --contract | --setting | --release TAG'
    ;;
esac
