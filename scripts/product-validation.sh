#!/usr/bin/env bash
set -euo pipefail

output_dir=${1:?output directory is required}
report="$output_dir/conformance-report.json"

jq -e '
  (.cells == 12) and (.paired_indicator_vector | length == 12) and
  (.proof_choice_counts == {FOUNDATION:4,COHERENCE:4,REGRESSION:4}) and
  (.indicator_class_counts == {DRIVER:4,OUTCOME:4,GUARDRAIL:4}) and
  (.unknown_records | length == 4) and
  (.repository_writes == 0) and (.local_test_executions == 0) and
  (.cross_project_required_gates == 0) and
  ([.paired_indicator_vector[] | select(contains("="))] | length) == 12
' "$report" >/dev/null

jq -e '
  .denominator == 12 and (.cases | length == 12) and
  ([.proof_choices[] | select(.name == "FOUNDATION" and .total == 4 and .closed == 4 and .unknown == 0 and .refuted == 0)] | length) == 1 and
  ([.proof_choices[] | select(.name == "COHERENCE" and .total == 4 and .closed == 0 and .unknown == 4 and .refuted == 0)] | length) == 1 and
  ([.proof_choices[] | select(.name == "REGRESSION" and .total == 4 and .closed == 0 and .unknown == 0 and .refuted == 4)] | length) == 1 and
  ([.indicator_classes[] | select(.name == "DRIVER" and .total == 4)] | length) == 1 and
  ([.indicator_classes[] | select(.name == "OUTCOME" and .total == 4)] | length) == 1 and
  ([.indicator_classes[] | select(.name == "GUARDRAIL" and .total == 4)] | length) == 1 and
  (.improvement.status == "UNKNOWN" and .improvement.before == null and .improvement.after == null and .improvement.delta == null)
' "$output_dir/semantic-metrics-dossier.json" >/dev/null

find "$output_dir/cases" -type f -name '*.dossier.json' -exec jq -e '
  if .decision == "ACCEPT" then
    (.evaluation.identity_match == true and .evaluation.counterexample_removed == true and .evaluation.corpus_preserved == true and (.evaluation.evidence | any(.[]; .class == "counterexample" and .preserved == true)))
  else true end
' {} + >/dev/null

find "$output_dir/cases" -type f -name '*.dossier.json' -exec jq -e '(.proof_choice != "" and .indicator_class != "" and .evaluation.improvement.status == "UNKNOWN" and .evaluation.improvement.before == null and .evaluation.improvement.after == null and .evaluation.improvement.delta == null)' {} + >/dev/null

echo 'product_validation=CLOSED'
