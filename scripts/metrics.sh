#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
output_dir=${1:?output directory is required}
runtime_file=${2:?runtime file is required}
report="$output_dir/conformance-report.json"

go_files=$(find "$repo_root" -type f -name '*.go' ! -path "$repo_root/.git/*" | wc -l | tr -d ' ')
gooo_files=$(find "$repo_root" -type f -name '*.gooo' ! -path "$repo_root/.git/*" | wc -l | tr -d ' ')
physical_lines=$(find "$repo_root" -type f \( -name '*.go' -o -name '*.gooo' \) ! -path "$repo_root/.git/*" -exec wc -l {} + | awk 'END {print $1+0}')
descendant_dirs=$(find "$repo_root" -mindepth 1 -type d ! -path "$repo_root/.git" ! -path "$repo_root/.git/*" | wc -l | tr -d ' ')
regular_files=$(find "$repo_root" -type f ! -path "$repo_root/.git/*" ! -path "$repo_root/README.md" | wc -l | tr -d ' ')
generated_files=$(find "$output_dir" -type f | wc -l | tr -d ' ')
generated_bytes=$(find "$output_dir" -type f -exec wc -c {} + | awk 'END {print $1+0}')
wall_ms=$(awk '{printf "%d", $1 * 1000}' "$runtime_file")
peak_rss_kib=$(awk '{print $2+0}' "$runtime_file")
tests_total=$(jq '.scenarios' "$report")
tests_selected=$(jq '.scenarios' "$report")
tests_executed=$(jq '[.cases[] | select(.pass)] | length' "$report")
tests_reused=0
tests_failed=$(jq '[.cases[] | select(.pass | not)] | length' "$report")
tests_unknown=$(jq '.unknown' "$report")

jq -n \
  --arg schema "gooo/counterexample-guided-rewriter/metrics/v1" \
  --argjson go_files "$go_files" \
  --argjson gooo_files "$gooo_files" \
  --argjson physical_lines "$physical_lines" \
  --argjson descendant_dirs "$descendant_dirs" \
  --argjson regular_files "$regular_files" \
  --argjson generated_files "$generated_files" \
  --argjson generated_bytes "$generated_bytes" \
  --argjson wall_ms "$wall_ms" \
  --argjson peak_rss_kib "$peak_rss_kib" \
  --argjson tests_total "$tests_total" \
  --argjson tests_selected "$tests_selected" \
  --argjson tests_executed "$tests_executed" \
  --argjson tests_reused "$tests_reused" \
  --argjson tests_failed "$tests_failed" \
  --argjson tests_unknown "$tests_unknown" \
  '{schema:$schema,go_files:$go_files,gooo_files:$gooo_files,physical_lines:$physical_lines,descendant_dirs:$descendant_dirs,regular_files:$regular_files,generated_files:$generated_files,generated_bytes:$generated_bytes,wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,tests_total:$tests_total,tests_selected:$tests_selected,tests_executed:$tests_executed,tests_reused:$tests_reused,tests_failed:$tests_failed,tests_unknown:$tests_unknown}' \
  > "$output_dir/metrics.json"

cat "$output_dir/metrics.json"
