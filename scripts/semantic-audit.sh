#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
meta="$repo_root/.gooo/rewrite.gooo"

test "$(grep -c '^operator ' "$meta")" -eq 3
test "$(grep -c '^scenario ' "$meta")" -eq 12
test "$(grep -c '^meta_activity ' "$meta")" -eq 12
test "$(grep -c 'proof_choice=' "$meta")" -eq 12
test "$(grep -c 'indicator_class=' "$meta")" -eq 12
test "$(grep -c '^predicate ' "$meta")" -eq 10
grep -q '^authority metacode$' "$meta"
grep -q '^search bound=1 ' "$meta"
grep -q '^cross_project_gate required=0$' "$meta"
grep -q '^source_policy .*repository_writes=zero .*local_test_executions=zero .*auto_apply=forbidden .*git_integration=forbidden$' "$meta"
grep -q '^unknown_fields stage,step,reason,unknown_class,next_operation,blocked_by$' "$meta"
grep -q '^candidate_space .*bound=1$' "$meta"
grep -q '^acceptance relation=counterexample-removed-and-corpus-unchanged ' "$meta"
grep -q '^proof_counts closed=4 unknown=4 refuted=4$' "$meta"
grep -q '^indicator_counts closed=4 unknown=4 refuted=4$' "$meta"
test "$(find "$repo_root/fixtures/cases" -type f -name '*.json' | wc -l | tr -d ' ')" -eq 12
if grep -R -n 'CONTRACT_DIGEST_PLACEHOLDER' "$repo_root/fixtures"; then
	echo 'fixture contract identity is not pinned' >&2
	exit 1
fi
grep -q -- '--draft' "$repo_root/.github/workflows/release.yml"
grep -q 'verify-release-lineage.sh' "$repo_root/.github/workflows/release.yml"
test -f "$repo_root/docs/operational-incidents.json"
grep -q 'PR_FIRST_BYPASSED' "$repo_root/docs/operational-incidents.json"
test -f "$repo_root/docs/operational-incident-tag-reuse-receipt.json"
grep -q 'TAG_DELETED_AND_VERSION_REUSED' "$repo_root/docs/operational-incident-tag-reuse-receipt.json"
if grep -nE 'release delete|tag -f|--force|--clobber' "$repo_root/.github/workflows"; then
	echo 'public release deletion, retagging, or overwrite is forbidden' >&2
	exit 1
fi
grep -q 'README.md' "$repo_root/contracts/metrics-v1.json"
if grep -nE 'FIXED_POINT|score|percentage' "$repo_root/cmd" "$repo_root/.gooo"; then
	echo 'implicit fixed-point or aggregate metric is forbidden' >&2
	exit 1
fi
if grep -nE 'git (commit|merge|push|reset|checkout)|gh (pr merge|release delete|release edit)' "$repo_root/cmd" "$repo_root/internal"; then
	echo 'automatic source mutation or repository integration is forbidden' >&2
  exit 1
fi
echo 'semantic_audit=CLOSED'
