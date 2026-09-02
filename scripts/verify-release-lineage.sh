#!/usr/bin/env bash
set -euo pipefail

repo=${RELEASE_LINEAGE_REPO:-kimjooyoon/gooo-counterexample-guided-rewriter}
tag=${1:-${GITHUB_REF_NAME:-}}
api_version=2026-03-10

fail_closed() {
	echo "release lineage guard: $1" >&2
	exit 1
}

[[ -n "${GH_TOKEN:-}" ]] || fail_closed 'GH_TOKEN is required'
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail_closed "invalid release tag: $tag"

tag_payload=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repo/git/ref/tags/$tag") || fail_closed "tag $tag cannot be read"
object_type=$(jq -r 'if type == "object" and .object.type then .object.type else "missing" end' <<<"$tag_payload") || fail_closed 'tag response is not valid JSON'
commit_sha=$(jq -r 'if type == "object" and .object.sha then .object.sha else "missing" end' <<<"$tag_payload") || fail_closed 'tag response is not valid JSON'

if [[ "$object_type" == 'tag' ]]; then
	tag_object=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repo/git/tags/$commit_sha") || fail_closed "annotated tag $tag cannot be dereferenced"
	commit_sha=$(jq -r 'if type == "object" and .object.sha then .object.sha else "missing" end' <<<"$tag_object") || fail_closed 'annotated tag response is not valid JSON'
	object_type=$(jq -r 'if type == "object" and .object.type then .object.type else "missing" end' <<<"$tag_object") || fail_closed 'annotated tag response is not valid JSON'
fi

[[ "$object_type" == 'commit' && "$commit_sha" != 'missing' ]] || fail_closed "tag $tag does not resolve to a commit"

compare_payload=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repo/compare/main...$commit_sha") || fail_closed "main lineage for $commit_sha cannot be read"
compare_status=$(jq -r 'if type == "object" and .status then .status else "missing" end' <<<"$compare_payload") || fail_closed 'compare response is not valid JSON'
ahead_by=$(jq -r 'if type == "object" and .ahead_by then .ahead_by else -1 end' <<<"$compare_payload") || fail_closed 'compare response is not valid JSON'
[[ ("$compare_status" == 'behind' || "$compare_status" == 'identical') && "$ahead_by" -eq 0 ]] || fail_closed "tag $tag commit $commit_sha is not in main lineage"

commit_payload=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repo/commits/$commit_sha") || fail_closed "commit $commit_sha cannot be read"
commit_message=$(jq -r 'if type == "object" and .commit.message then .commit.message else "missing" end' <<<"$commit_payload") || fail_closed 'commit response is not valid JSON'
pr_number=$(sed -nE 's/.*#([0-9]+).*/\1/p' <<<"$commit_message" | tail -n 1)

lineage_source='commit_message_pr'
merged_pr='false'
observed_base='missing'
observed_merge_commit='missing'
observed_merged_at='missing'
if [[ -n "$pr_number" ]]; then
	pr_view_payload=$(gh pr view "$pr_number" --json number,mergedAt,baseRefName,mergeCommit) || fail_closed "PR #$pr_number cannot be read"
	observed_base=$(jq -r '.baseRefName // "missing"' <<<"$pr_view_payload") || fail_closed 'PR view response is not valid JSON'
	observed_merge_commit=$(jq -r '.mergeCommit.oid // "missing"' <<<"$pr_view_payload") || fail_closed 'PR view response is not valid JSON'
	observed_merged_at=$(jq -r '.mergedAt // "missing"' <<<"$pr_view_payload") || fail_closed 'PR view response is not valid JSON'
	merged_pr=$(jq -r --arg sha "$commit_sha" 'if type == "object" then ((.mergedAt != null and .baseRefName == "main" and .mergeCommit.oid == $sha) | tostring) else "missing" end' <<<"$pr_view_payload") || fail_closed 'PR view response is not valid JSON'
fi

if [[ "$merged_pr" != 'true' ]]; then
	lineage_source='commit_pulls'
	pulls_payload=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repo/commits/$commit_sha/pulls") || fail_closed "PR lineage for $commit_sha cannot be read"
	merged_pr=$(jq -r --arg sha "$commit_sha" 'if type == "array" then (any(.[]; .merged_at != null and .base.ref == "main" and .merge_commit_sha == $sha) | tostring) else "missing" end' <<<"$pulls_payload") || fail_closed 'PR lineage response is not valid JSON'
fi

if [[ "$merged_pr" != 'true' ]]; then
	lineage_source='closed_main_pulls'
	pulls_payload=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repo/pulls?state=closed&base=main&per_page=100") || fail_closed 'closed main PR lineage cannot be read'
	merged_pr=$(jq -r --arg sha "$commit_sha" 'if type == "array" then (any(.[]; .merged_at != null and .base.ref == "main" and .merge_commit_sha == $sha) | tostring) else "missing" end' <<<"$pulls_payload") || fail_closed 'closed PR response is not valid JSON'
fi

[[ "$merged_pr" == 'true' ]] || fail_closed "tag $tag commit $commit_sha has no merged PR lineage into main pr_number=${pr_number:-unknown} source=$lineage_source observed_base=$observed_base observed_merge_commit=$observed_merge_commit observed_merged_at=$observed_merged_at"

echo "release_lineage=CLOSED tag=$tag commit=$commit_sha merged_pr_into_main=true pr_number=${pr_number:-unknown} source=$lineage_source"
