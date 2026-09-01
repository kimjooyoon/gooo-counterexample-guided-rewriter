#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
output_dir=${1:-"$(mktemp -d)"}
runtime_file=$(mktemp)
trap 'rm -f "$runtime_file"' EXIT

mkdir -p "$output_dir"
/usr/bin/time -f '%e %M' -o "$runtime_file" go run ./cmd/gooo-counterexample-guided-rewriter -meta .gooo/rewrite.gooo -input-root "$repo_root" -fixed -output "$output_dir"
jq -e '.schema == "gooo/counterexample-guided-rewriter/conformance/v1" and .scenarios == 7 and .closed == 4 and .unknown == 2 and .refuted == 1 and ([.cases[] | select(.pass == false)] | length) == 0 and .repository_writes == 0' "$output_dir/conformance-report.json" >/dev/null
"$repo_root/scripts/metrics.sh" "$output_dir" "$runtime_file" > "$output_dir/metrics.stdout.json"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo '### fixed conformance'
    jq -r '"decision=" + .decision + " scenarios=" + (.scenarios|tostring) + " closed=" + (.closed|tostring) + " unknown=" + (.unknown|tostring) + " refuted=" + (.refuted|tostring) + " repository_writes=" + (.repository_writes|tostring)' "$output_dir/conformance-report.json"
    echo '### exact inventory and test metrics'
    jq -r 'to_entries | map(.key + "=" + (.value|tostring)) | join(" ")' "$output_dir/metrics.json"
  } >> "$GITHUB_STEP_SUMMARY"
fi
