#!/usr/bin/env bash
set -euo pipefail

output_dir=${1:?output directory is required}
report="$output_dir/conformance-report.json"

jq -e '
  (.cells == 12) and (.paired_indicator_vector | length == 12) and
  (.repository_writes == 0) and (.local_test_executions == 0) and
  (.cross_project_required_gates == 0) and
  ([.paired_indicator_vector[] | select(contains("="))] | length) == 12
' "$report" >/dev/null

find "$output_dir/cases" -type f -name '*.dossier.json' -exec jq -e '
  if .decision == "ACCEPT" then
    (.evaluation.identity_match == true and .evaluation.counterexample_removed == true and .evaluation.corpus_preserved == true and (.evaluation.evidence | any(.[]; .class == "counterexample" and .preserved == true)))
  else true end
' {} + >/dev/null

echo 'product_validation=CLOSED'
