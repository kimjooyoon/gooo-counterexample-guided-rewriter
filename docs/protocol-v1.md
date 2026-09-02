# Counterexample-guided rewrite protocol v1

## Input boundary

The input is a reducer-style `gooo.semantic_counterexample_reduction_report/v1` JSON record. The failing fixture may be `UNKNOWN` or `REFUTED`; `reduced_graph` must be a typed semantic graph. `source_digest`, `toolchain_digest`, `origin_source_digest`, and the contract/space/rule/evaluator identities identify the immutable source and evaluation context. The record carries a causal counterexample input, a target terminal, and normal/regression corpus pairs.

The loader additionally accepts the richer `gooo/counterexample-guided-rewriter/counterexample/v1` alias so callers can preserve explicit origin and replay evidence. Existing reducer reports remain the compatibility format.

## Candidate contract

Each operator produces a self-describing `.gooo` candidate, a typed IR artifact, a caller-owned patch artifact, and a candidate dossier. The patch is a proposal only: the runtime never applies it or writes to the input repository. A candidate is eligible only when all declared hard predicates are true:

1. the typed operator precondition is observed;
2. origin and toolchain digests remain equal to the input record;
3. the transformed IR digest is recomputable;
4. affected IDs are deterministic and non-empty;
5. terminal reason and effect trace equal `target_terminal`;
6. replay is stable for the declared two replays;
7. the causal counterexample is removed by the declared structural rule;
8. every normal and regression corpus input is unchanged and still matches its expected behavior;
9. the before/after fixture, source, contract, toolchain, and evaluator identities match exactly.

The candidate IR may change the terminal decision from the fixture's `UNKNOWN` or `REFUTED` baseline to the target `CLOSED` decision. “Reason preserving” therefore means preserving the declared repair contract, rather than copying a rejection reason that the rewrite is supposed to resolve. A candidate that removes the causal counterexample but changes a normal or regression behavior is a known contradiction and is `REFUTED`.

## Decision lattice

Candidate outcomes are folded using `REFUTED > UNKNOWN > CLOSED`. Known contradictions (including reason/effect drift, hidden-only repair, or changed corpus behavior) are `REFUTED` first. Missing candidate-space, rule, or evaluator identity is an `UNKNOWN` with exactly the six declared fields. No eligible candidate, an identity mismatch, missing origin/evidence, or more than one equally ranked eligible candidate is `UNKNOWN`. One and only one eligible candidate is dossier `ACCEPT` and case `CLOSED`.

## Bound and provenance

The search bound is one pass over exactly three finite rules in `.gooo` order. There is no mutation of source code and no automatic Git operation. Optional reducer/planner/compiler pins are provenance-only inputs; the required cross-project gate is zero. The twelve fixed cells are the sole denominator, with 4/4/4 case, proof, and indicator counts; aggregate scores and percentages are not part of the contract.
