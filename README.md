# gooo-counterexample-guided-rewriter

`gooo-counterexample-guided-rewriter` turns a reducer-style semantic counterexample into a bounded, deterministic set of typed IR rewrite candidates. The authoritative language is [`/.gooo/rewrite.gooo`](.gooo/rewrite.gooo); Go only parses, lowers, executes, and verifies that contract.

The first release contains three concrete operators:

- `guard-insertion`: inserts a typed guard immediately before a guardable effect;
- `effect-narrowing`: narrows an effect capability while retaining its observable effect trace;
- `reason-preserving-branch-split`: splits a typed branch into two routes while retaining the terminal reason and effect trace.

Every candidate records its preconditions, input and transformed IR digests, affected stable semantic IDs, expected terminal trace, replay result, and counterexample visibility. Search is bounded to one deterministic pass in the `.gooo` order. Exactly one candidate satisfying every hard predicate is `CLOSED`; a tie or insufficient evidence is `UNKNOWN`; a reason/effect drift or hidden counterexample is `REFUTED`.

## Run

The command writes only to a caller-owned output directory outside the input repository. It checks the input boundary before and after generation and reports `repository_writes=0` when unchanged.

```text
go run ./cmd/gooo-counterexample-guided-rewriter \
  -meta .gooo/rewrite.gooo \
  -input fixtures/cases/closed-guard-insertion.json \
  -input-root . \
  -output /tmp/gooo-rewriter-output
```

The fixed corpus is the seven scenarios declared in `.gooo`. CI runs the corpus, Go tests, vet, and conformance; local verification is intentionally not part of the release protocol.

The reducer `v0.1.1` and error-directed planner `v0.1.1` are recorded as immutable, digest-pinned optional oracles. They are never required for this repository's conformance gate (`cross_project_gate=0`). No source mutation, commit, merge, or input-repository output is performed by the rewriter.
