#!/usr/bin/env bash
set -euo pipefail

output_dir=${1:?output directory is required}
report="$output_dir/conformance-report.json"

jq -e '
  (has("score") | not) and (has("percentage") | not) and
  ([.cases[] | select(.pass == true)] | length) == 12 and
  ([.cases[] | select(.expected == "CLOSED" and .observed == "CLOSED")] | length) == 4 and
  ([.cases[] | select(.expected == "UNKNOWN" and .observed == "UNKNOWN")] | length) == 4 and
  ([.cases[] | select(.expected == "REFUTED" and .observed == "REFUTED")] | length) == 4
' "$report" >/dev/null

metrics="$output_dir/metrics.json"
if [[ -f "$metrics" ]]; then
  jq -e '.candidate_gooo_files == 36 and .candidate_ir_files == 36 and .candidate_patch_files == 36 and .candidate_dossier_files == 36 and .case_report_files == 12 and .causal_input_files == 12 and (.paired_indicator_vector | length) == 12 and (has("score") | not) and (has("percentage") | not)' "$metrics" >/dev/null
fi

find "$output_dir/cases" -type f -name '*.gooo' -exec jq -R -e 'length > 0' {} + >/dev/null
jq -e '.candidates | all(.[]; (.artifact_gooo != null and .artifact_ir != null and .artifact_patch != null and .artifact_dossier != null))' "$output_dir/cases"/*/case-report.json >/dev/null
find "$output_dir/cases" -type f -name '*.dossier.json' -exec jq -e '(.decision == "ACCEPT" or .decision == "UNKNOWN" or .decision == "REFUTED")' {} + >/dev/null
find "$output_dir/cases" -type f -name '*.patch.json' -exec jq -e '(.caller_owned == true and .auto_apply == false and .repository_writes == 0)' {} + >/dev/null

echo 'jq_assertions=CLOSED'
