#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
output_dir=${1:-"$(mktemp -d)"}
runtime_file=$(mktemp)
trap 'rm -f "$runtime_file"' EXIT

mkdir -p "$output_dir"
/usr/bin/time -f '%e %M' -o "$runtime_file" go run ./cmd/gooo-counterexample-guided-rewriter -meta .gooo/rewrite.gooo -input-root "$repo_root" -fixed -output "$output_dir"
jq -e '.schema == "gooo/counterexample-guided-rewriter/conformance/v1" and .scenarios == 12 and .cells == 12 and .closed == 4 and .unknown == 4 and .refuted == 4 and .proof_counts == {"CLOSED":4,"UNKNOWN":4,"REFUTED":4} and .indicator_counts == {"CLOSED":4,"UNKNOWN":4,"REFUTED":4} and .proof_choice_counts == {"FOUNDATION":4,"COHERENCE":4,"REGRESSION":4} and .indicator_class_counts == {"DRIVER":4,"OUTCOME":4,"GUARDRAIL":4} and (.unknown_records | length) == 4 and ([.unknown_records[] | select(.record.stage != "" and .record.step != "" and .record.reason != "" and .record.unknown_class != "" and .record.next_operation != "" and (.record.blocked_by | length > 0))] | length) == 4 and ([.cases[] | select(.pass == false)] | length) == 0 and (.paired_indicator_vector | length) == 12 and .external_utility_unknown == 1 and .repository_writes == 0 and .local_test_executions == 0 and .cross_project_required_gates == 0' "$output_dir/conformance-report.json" >/dev/null
jq -e '.schema == "gooo/counterexample-guided-rewriter/semantic-metrics-dossier/v1" and .denominator == 12 and (.cases | length) == 12 and (.unknown_records | length) == 4 and ([.cases[] | select(.proof_choice != "" and .indicator_class != "")] | length) == 12 and ([.proof_choices[] | select(.name == "FOUNDATION" and .total == 4)] | length) == 1 and ([.proof_choices[] | select(.name == "COHERENCE" and .total == 4)] | length) == 1 and ([.proof_choices[] | select(.name == "REGRESSION" and .total == 4)] | length) == 1 and ([.indicator_classes[] | select(.name == "DRIVER" and .total == 4)] | length) == 1 and ([.indicator_classes[] | select(.name == "OUTCOME" and .total == 4)] | length) == 1 and ([.indicator_classes[] | select(.name == "GUARDRAIL" and .total == 4)] | length) == 1 and (.improvement.status == "UNKNOWN" and .improvement.before == null and .improvement.after == null and .improvement.delta == null)' "$output_dir/semantic-metrics-dossier.json" >/dev/null
"$repo_root/scripts/product-validation.sh" "$output_dir"
"$repo_root/scripts/metrics.sh" "$output_dir" "$runtime_file" > "$output_dir/metrics.stdout.json"
"$repo_root/scripts/assertions.sh" "$output_dir"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo '### fixed conformance'
    jq -r '"decision=" + .decision + " cells=" + (.cells|tostring) + " scenarios=" + (.scenarios|tostring) + " closed=" + (.closed|tostring) + " unknown=" + (.unknown|tostring) + " refuted=" + (.refuted|tostring) + " proof=" + ([.proof_counts.CLOSED,.proof_counts.UNKNOWN,.proof_counts.REFUTED] | map(tostring) | join("/")) + " indicators=" + ([.indicator_counts.CLOSED,.indicator_counts.UNKNOWN,.indicator_counts.REFUTED] | map(tostring) | join("/")) + " external_utility_unknown=" + (.external_utility_unknown|tostring) + " repository_writes=" + (.repository_writes|tostring) + " local_test_executions=" + (.local_test_executions|tostring) + " cross_project_required_gates=" + (.cross_project_required_gates|tostring)' "$output_dir/conformance-report.json"
    echo '### exact inventory and test metrics'
    jq -r 'to_entries | map(.key + "=" + (.value|tostring)) | join(" ")' "$output_dir/metrics.json"
  } >> "$GITHUB_STEP_SUMMARY"
fi
