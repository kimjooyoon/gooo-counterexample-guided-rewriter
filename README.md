# gooo-counterexample-guided-rewriter

`gooo-counterexample-guided-rewriter` closes a small counterexample-guided self-improvement loop without changing its input repository. The authoritative language is [`/.gooo/rewrite.gooo`](.gooo/rewrite.gooo); Go only parses, lowers, enumerates, evaluates, and verifies that contract.

The first release contains three concrete operators:

- `guard-insertion`: inserts a typed guard immediately before a guardable effect;
- `effect-narrowing`: narrows an effect capability while retaining its observable effect trace;
- `reason-preserving-branch-split`: splits a typed branch into two routes while retaining the terminal reason and effect trace.

Every candidate records its causal input, preconditions, input and transformed IR digests, affected stable semantic IDs, expected terminal trace, replay result, paired before/after evaluation, and corpus evidence. Search is bounded to one deterministic pass in the `.gooo` order. Exactly one candidate satisfying the acceptance relation is candidate `ACCEPT` and case `CLOSED`; a tie or missing identity/evidence is `UNKNOWN`; a known contradiction, hidden causal input, or normal/regression behavior change is `REFUTED`.

The fixed contract has exactly twelve cells: `4 CLOSED`, `4 UNKNOWN`, and `4 REFUTED`. It also declares `4/4/4` proof and indicator counts. The compiler fixture `closed-branch-split` starts with an `UNKNOWN` top decision and closes it only through an explicit `explicit_fail_closed` branch; no implicit `FIXED_POINT` state is used.

## Run

The command writes only to a caller-owned output directory outside the input repository. It checks the input boundary before and after generation and reports `repository_writes=0` when unchanged.

```text
go run ./cmd/gooo-counterexample-guided-rewriter \
  -meta .gooo/rewrite.gooo \
  -input fixtures/cases/closed-guard-insertion.json \
  -input-root . \
  -output /tmp/gooo-rewriter-output
```

The fixed corpus is the twelve scenarios declared in `.gooo`. CI on pull requests and the default branch performs all generation and paired before/after validation, then publishes caller-owned artifacts. Local test, build, vet, formatting, shell, action, assertion, generator, conformance, and product-validation executions are deliberately absent from the release protocol.

The reducer `v0.1.1` and error-directed planner `v0.1.1` are recorded as immutable, digest-pinned optional oracles. The bounded self-change compiler and causal counterexample reducer are optional immutable-release inputs only; they are never required for this repository's conformance gate (`cross_project_gate=0`). No source mutation, auto-apply, commit, merge, or input-repository output is performed by the runtime.
