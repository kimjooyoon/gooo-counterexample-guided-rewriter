#!/usr/bin/env bash
set -euo pipefail

output_dir=${1:?generated output directory is required}
actions_metrics=${2:?Actions metrics JSON is required}
incident_receipt=${3:?historical incident receipt is required}
release_tag=${4:-${GITHUB_REF_NAME:-unknown}}
release_commit=${5:-${GITHUB_SHA:-unknown}}

conformance="$output_dir/conformance-report.json"
semantic="$output_dir/semantic-metrics-dossier.json"
metrics="$output_dir/metrics.json"

for required in "$conformance" "$semantic" "$metrics" "$actions_metrics" "$incident_receipt"; do
	if [[ ! -f "$required" ]]; then
		echo "release evidence: missing $required" >&2
		exit 1
	fi
done

jq -e '
	.schema == "gooo/counterexample-guided-rewriter/conformance/v1" and
	.scenarios == 12 and .cells == 12 and .closed == 4 and .unknown == 4 and .refuted == 4 and
	.proof_choice_counts == {FOUNDATION:4,COHERENCE:4,REGRESSION:4} and
	.indicator_class_counts == {DRIVER:4,OUTCOME:4,GUARDRAIL:4} and
	([.unknown_records[] | select(.record.stage != "" and .record.step != "" and .record.reason != "" and .record.unknown_class != "" and .record.next_operation != "" and (.record.blocked_by | length > 0))] | length) == 4
' "$conformance" >/dev/null

jq -e '
	.schema == "gooo/counterexample-guided-rewriter/semantic-metrics-dossier/v1" and
	.denominator == 12 and (.cases | length) == 12 and (.unknown_records | length) == 4 and
	([.cases[] | select(.proof_choice != "" and .indicator_class != "")] | length) == 12 and
	([.proof_choices[] | select(.name == "FOUNDATION" and .total == 4 and .closed == 4 and .unknown == 0 and .refuted == 0)] | length) == 1 and
	([.proof_choices[] | select(.name == "COHERENCE" and .total == 4 and .closed == 0 and .unknown == 4 and .refuted == 0)] | length) == 1 and
	([.proof_choices[] | select(.name == "REGRESSION" and .total == 4 and .closed == 0 and .unknown == 0 and .refuted == 4)] | length) == 1 and
	([.indicator_classes[] | select(.name == "DRIVER" and .total == 4)] | length) == 1 and
	([.indicator_classes[] | select(.name == "OUTCOME" and .total == 4)] | length) == 1 and
	([.indicator_classes[] | select(.name == "GUARDRAIL" and .total == 4)] | length) == 1
' "$semantic" >/dev/null

jq -e '
	.status == "OBSERVED" and
	(.build_ms | type == "number") and (.test_ms | type == "number") and (.wall_ms | type == "number") and (.peak_rss_kib | type == "number") and
	((.cache_hits == null) or (.cache_hits | type == "number")) and ((.cache_misses == null) or (.cache_misses | type == "number")) and
	((.cache_hits != null and .cache_misses != null) or (.cache_unknown.status == "UNKNOWN" and .cache_unknown.stage != "" and .cache_unknown.step != "" and .cache_unknown.reason != "" and .cache_unknown.unknown_class != "" and .cache_unknown.next_operation != "" and (.cache_unknown.blocked_by | length > 0)))
' "$actions_metrics" >/dev/null

updated_metrics="$output_dir/metrics.with-actions.next.json"
jq --slurpfile actions "$actions_metrics" '. + {
	build_ms: $actions[0].build_ms,
	test_ms: $actions[0].test_ms,
	wall_ms: $actions[0].wall_ms,
	peak_rss_kib: $actions[0].peak_rss_kib,
	cache_hits: $actions[0].cache_hits,
	cache_misses: $actions[0].cache_misses,
	actions_run_id: $actions[0].run_id,
	metrics_status: $actions[0].status,
	metrics_unknown: $actions[0].cache_unknown
}' "$metrics" > "$updated_metrics"
mv "$updated_metrics" "$metrics"

jq -S -n \
	--arg tag "$release_tag" \
	--arg commit "$release_commit" \
	--slurpfile conformance "$conformance" \
	--slurpfile semantic "$semantic" \
	--slurpfile metrics "$metrics" \
	--slurpfile actions "$actions_metrics" \
	--slurpfile incident "$incident_receipt" \
	'{
		schema: "gooo/counterexample-guided-rewriter/release-evidence/v1",
		authority: "github-actions",
		release: {tag: $tag, commit: $commit, run_id: ($actions[0].run_id // null)},
		historical_preservation: {
			v0_1_3: {tag: "v0.1.3", release_id: 381184181, immutable: true, commit: "12f22e92fded26a12321001f82cbbc84f154c620"},
			tag_reuse_incident: $incident[0]
		},
		conformance: $conformance[0],
		semantic_metrics: $semantic[0],
		metrics: $metrics[0],
		actions_metrics: $actions[0],
		case_mappings: ($semantic[0].cases | map({scenario,proof_choice,indicator_class,expected,observed,pass})),
		unknown_records: $semantic[0].unknown_records,
		improvement: $semantic[0].improvement,
		runtime_contract: {repository_writes: 0, local_test_executions: 0, cross_project_required_gates: 0}
	}' > "$output_dir/release-evidence.json"

echo "release_evidence=$output_dir/release-evidence.json"
