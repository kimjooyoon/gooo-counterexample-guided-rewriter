#!/usr/bin/env bash
set -euo pipefail

output_dir=${1:?output directory is required}
report="$output_dir/conformance-report.json"

jq -e '
  (has("score") | not) and (has("percentage") | not) and
  ([.cases[] | select(.pass == true)] | length) == 12 and
  ([.cases[] | select(.expected == "CLOSED" and .observed == "CLOSED")] | length) == 4 and
  ([.cases[] | select(.expected == "UNKNOWN" and .observed == "UNKNOWN")] | length) == 4 and
  ([.cases[] | select(.expected == "REFUTED" and .observed == "REFUTED")] | length) == 4 and
  .proof_choice_counts == {FOUNDATION:4,COHERENCE:4,REGRESSION:4} and
  .indicator_class_counts == {DRIVER:4,OUTCOME:4,GUARDRAIL:4} and
  ([.unknown_records[] | select(.record.stage != "" and .record.step != "" and .record.reason != "" and .record.unknown_class != "" and .record.next_operation != "" and (.record.blocked_by | length > 0))] | length) == 4 and
  ([.cases[] | select(.proof_choice != "" and .indicator_class != "")] | length) == 12
' "$report" >/dev/null

metrics="$output_dir/metrics.json"
if [[ -f "$metrics" ]]; then
  jq -e '.candidate_gooo_files == 36 and .candidate_ir_files == 36 and .candidate_patch_files == 36 and .candidate_dossier_files == 36 and .case_report_files == 12 and .causal_input_files == 12 and (.paired_indicator_vector | length) == 12 and (.semantic_metrics_dossier_files == 1) and (has("score") | not) and (has("percentage") | not)' "$metrics" >/dev/null
fi

jq -e '.schema == "gooo/counterexample-guided-rewriter/semantic-metrics-dossier/v1" and .denominator == 12 and (.cases | length) == 12 and (.unknown_records | length) == 4 and (.improvement.status == "UNKNOWN" and .improvement.before == null and .improvement.after == null and .improvement.delta == null)' "$output_dir/semantic-metrics-dossier.json" >/dev/null

find "$output_dir/cases" -type f -name '*.gooo' -exec jq -R -e 'length > 0' {} + >/dev/null
jq -e '.candidates | all(.[]; (.artifact_gooo != null and .artifact_ir != null and .artifact_patch != null and .artifact_dossier != null))' "$output_dir/cases"/*/case-report.json >/dev/null
find "$output_dir/cases" -type f -name '*.dossier.json' -exec jq -e '(.decision == "ACCEPT" or .decision == "UNKNOWN" or .decision == "REFUTED")' {} + >/dev/null
find "$output_dir/cases" -type f -name '*.patch.json' -exec jq -e '(.caller_owned == true and .auto_apply == false and .repository_writes == 0)' {} + >/dev/null

echo 'jq_assertions=CLOSED'
