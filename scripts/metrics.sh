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
candidate_gooo_files=$(find "$output_dir" -type f -name '*.gooo' | wc -l | tr -d ' ')
candidate_ir_files=$(find "$output_dir" -type f -name '*.ir.json' | wc -l | tr -d ' ')
candidate_patch_files=$(find "$output_dir" -type f -name '*.patch.json' | wc -l | tr -d ' ')
candidate_dossier_files=$(find "$output_dir" -type f -name '*.dossier.json' | wc -l | tr -d ' ')
case_report_files=$(find "$output_dir" -type f -name 'case-report.json' | wc -l | tr -d ' ')
causal_input_files=$(find "$output_dir" -type f -name 'causal-input.json' | wc -l | tr -d ' ')
semantic_metrics_dossier_files=$(find "$output_dir" -type f -name 'semantic-metrics-dossier.json' | wc -l | tr -d ' ')
wall_ms=$(awk '{printf "%d", $1 * 1000}' "$runtime_file")
peak_rss_kib=$(awk '{print $2+0}' "$runtime_file")
tests_total=$(jq '.scenarios' "$report")
tests_selected=$(jq '.scenarios' "$report")
tests_executed=$(jq '[.cases[] | select(.pass)] | length' "$report")
tests_reused=0
tests_failed=$(jq '[.cases[] | select(.pass | not)] | length' "$report")
tests_unknown=$(jq '.unknown' "$report")
proof_closed=$(jq '.proof_counts.CLOSED' "$report")
proof_unknown=$(jq '.proof_counts.UNKNOWN' "$report")
proof_refuted=$(jq '.proof_counts.REFUTED' "$report")
indicator_closed=$(jq '.indicator_counts.CLOSED' "$report")
indicator_unknown=$(jq '.indicator_counts.UNKNOWN' "$report")
indicator_refuted=$(jq '.indicator_counts.REFUTED' "$report")
external_utility_unknown=$(jq '.external_utility_unknown' "$report")
local_test_executions=$(jq '.local_test_executions' "$report")
cross_project_required_gates=$(jq '.cross_project_required_gates' "$report")
paired_indicator_vector=$(jq -c '.paired_indicator_vector' "$report")

jq -n \
  --arg schema "gooo/counterexample-guided-rewriter/metrics/v1" \
  --argjson inventory_excludes '["README.md",".git"]' \
  --argjson go_files "$go_files" \
  --argjson gooo_files "$gooo_files" \
  --argjson physical_lines "$physical_lines" \
  --argjson descendant_dirs "$descendant_dirs" \
  --argjson regular_files "$regular_files" \
  --argjson generated_files "$generated_files" \
  --argjson generated_bytes "$generated_bytes" \
  --argjson candidate_gooo_files "$candidate_gooo_files" \
  --argjson candidate_ir_files "$candidate_ir_files" \
  --argjson candidate_patch_files "$candidate_patch_files" \
  --argjson candidate_dossier_files "$candidate_dossier_files" \
  --argjson case_report_files "$case_report_files" \
  --argjson causal_input_files "$causal_input_files" \
  --argjson semantic_metrics_dossier_files "$semantic_metrics_dossier_files" \
  --argjson wall_ms "$wall_ms" \
  --argjson peak_rss_kib "$peak_rss_kib" \
  --argjson tests_total "$tests_total" \
  --argjson tests_selected "$tests_selected" \
  --argjson tests_executed "$tests_executed" \
  --argjson tests_reused "$tests_reused" \
  --argjson tests_failed "$tests_failed" \
  --argjson tests_unknown "$tests_unknown" \
  --argjson proof_closed "$proof_closed" \
  --argjson proof_unknown "$proof_unknown" \
  --argjson proof_refuted "$proof_refuted" \
  --argjson indicator_closed "$indicator_closed" \
  --argjson indicator_unknown "$indicator_unknown" \
  --argjson indicator_refuted "$indicator_refuted" \
  --argjson external_utility_unknown "$external_utility_unknown" \
  --argjson local_test_executions "$local_test_executions" \
  --argjson cross_project_required_gates "$cross_project_required_gates" \
  --argjson paired_indicator_vector "$paired_indicator_vector" \
  --argjson build_ms null \
  --argjson test_ms null \
  --argjson cache_hits null \
  --argjson cache_misses null \
  --arg metrics_status 'UNKNOWN' \
  --argjson metrics_unknown '{"stage":"ACTIONS","step":"CAPTURE_ACTIONS_METRICS","reason":"actual Actions build/test/cache metrics are not available to this generation step","unknown_class":"ACTIONS_METRICS_UNAVAILABLE","next_operation":"RECORD_ACTIONS_METRICS","blocked_by":["actions-run"]}' \
  '{schema:$schema,inventory_excludes:$inventory_excludes,go_files:$go_files,gooo_files:$gooo_files,physical_lines:$physical_lines,descendant_dirs:$descendant_dirs,regular_files:$regular_files,generated_files:$generated_files,generated_bytes:$generated_bytes,candidate_gooo_files:$candidate_gooo_files,candidate_ir_files:$candidate_ir_files,candidate_patch_files:$candidate_patch_files,candidate_dossier_files:$candidate_dossier_files,case_report_files:$case_report_files,causal_input_files:$causal_input_files,semantic_metrics_dossier_files:$semantic_metrics_dossier_files,wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,build_ms:$build_ms,test_ms:$test_ms,cache_hits:$cache_hits,cache_misses:$cache_misses,metrics_status:$metrics_status,metrics_unknown:$metrics_unknown,tests_total:$tests_total,tests_selected:$tests_selected,tests_executed:$tests_executed,tests_reused:$tests_reused,tests_failed:$tests_failed,tests_unknown:$tests_unknown,proof_closed:$proof_closed,proof_unknown:$proof_unknown,proof_refuted:$proof_refuted,indicator_closed:$indicator_closed,indicator_unknown:$indicator_unknown,indicator_refuted:$indicator_refuted,external_utility_unknown:$external_utility_unknown,local_test_executions:$local_test_executions,cross_project_required_gates:$cross_project_required_gates,paired_indicator_vector:$paired_indicator_vector}' \
  > "$output_dir/metrics.json"

cat "$output_dir/metrics.json"
