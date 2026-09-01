# Counterexample-guided rewrite protocol v1

## Input boundary

The input is a reducer-style `gooo.semantic_counterexample_reduction_report/v1` JSON record. The `baseline` must be `REFUTED`, and `reduced_graph` must be a typed semantic graph. `source_digest`, `toolchain_digest`, and `origin_source_digest` identify the immutable source. The record also carries `target_terminal`, which is the terminal reason/effect contract a valid repair must produce.

The loader additionally accepts the richer `gooo/counterexample-guided-rewriter/counterexample/v1` alias so callers can preserve explicit origin and replay evidence. Existing reducer reports remain the compatibility format.

## Candidate contract

Each operator produces a self-describing `.gooo` candidate and a JSON semantic IR artifact. Candidate output is caller-owned. A candidate is eligible only when all declared hard predicates are true:

1. the typed operator precondition is observed;
2. origin and toolchain digests remain equal to the input record;
3. the transformed IR digest is recomputable;
4. affected IDs are deterministic and non-empty;
5. terminal reason and effect trace equal `target_terminal`;
6. replay is stable for the declared two replays;
7. the counterexample anchor remains visible.

The candidate IR may change the terminal decision from the reducer's `REFUTED` baseline to the target `CLOSED` decision. “Reason preserving” therefore means preserving the declared repair contract, rather than copying a rejection reason that the rewrite is supposed to resolve.

## Decision lattice

Candidate outcomes are folded using `REFUTED > UNKNOWN > CLOSED`. A reason/effect mismatch or deleted counterexample anchor is a direct `REFUTED`. No eligible candidate, a missing origin/evidence, or more than one equally ranked eligible candidate is `UNKNOWN`. One and only one eligible candidate is `CLOSED`.

## Bound and provenance

The search bound is one pass over exactly three operators in `.gooo` order. There is no mutation of source code and no automatic Git operation. Optional reducer/planner pins are provenance only; the required cross-project gate is zero.
