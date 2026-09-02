package rewriter

import "strings"

// EvaluateBeforeAfter is intentionally separate from the rewrite generator.
// It evaluates the same causal input and normal/regression corpus against two
// immutable IR snapshots and refuses to accept a candidate on target-terminal
// evidence alone.
func EvaluateBeforeAfter(meta MetaContract, c Counterexample, operator OperatorDecl, before, after SemanticIR, removed bool) EvaluationResult {
	identity := EvaluationIdentity{
		Fixture: c.Scenario, SourceDigest: c.SourceDigest, ContractDigest: c.ContractDigest,
		ToolchainDigest: c.ToolchainDigest, EvaluatorID: c.EvaluatorID, EvaluatorDigest: c.EvaluatorDigest,
	}
	result := EvaluationResult{
		Schema: EvaluationSchema, Identity: identity, IdentityMatch: exactEvaluationIdentity(meta, c, before, after),
		CausalInputID: c.CausalInput.ID, CounterexampleRemoved: removed, CorpusPreserved: true,
		Evidence: []CorpusEvidence{}, PairedIndicator: []string{},
	}
	if !result.IdentityMatch {
		result.Unknown = unknown("evaluate", "pair_before_after", "before and after do not share the exact fixture, source, contract, toolchain, and evaluator identity", "IDENTITY_MISMATCH", "recreate_paired_evaluation", "fixture-source-contract-toolchain-evaluator")
		return result
	}

	causal := CorpusCase{ID: c.CausalInput.ID, Class: "counterexample", Input: c.CausalInput.ID, Before: c.CausalInput.ObservedTerminal, After: c.CausalInput.TargetTerminal}
	causalBefore := evaluateCorpusInput(before, c, causal, false)
	causalAfter := evaluateCorpusInput(after, c, causal, true)
	causalExpectedBefore := causal.Before.Normalized()
	causalExpectedAfter := causal.After.Normalized()
	causalPairOK := sameTerminal(causalBefore, causalExpectedBefore) && sameTerminal(causalAfter, causalExpectedAfter)
	result.Evidence = append(result.Evidence, CorpusEvidence{
		InputID: causal.ID, Class: causal.Class, Before: causalBefore, After: causalAfter,
		ExpectedBefore: causalExpectedBefore, ExpectedAfter: causalExpectedAfter,
		Unchanged: sameTerminal(causalBefore, causalAfter), Preserved: removed && causalPairOK,
	})
	result.PairedIndicator = append(result.PairedIndicator, causal.ID+"="+causalBefore.Decision+"->"+causalAfter.Decision+":"+indicatorValue(removed && causalPairOK))
	result.CounterexampleRemoved = removed && causalPairOK && !causalAfter.CounterexampleVisible
	if !causalPairOK {
		result.KnownContradiction = true
		result.Contradiction = "candidate does not match the causal counterexample contract"
	}

	for _, corpus := range c.Corpus {
		beforeTrace := evaluateCorpusInput(before, c, corpus, false)
		afterTrace := evaluateCorpusInput(after, c, corpus, true)
		expectedBefore := corpus.Before.Normalized()
		expectedAfter := corpus.After.Normalized()
		unchanged := sameTerminal(beforeTrace, afterTrace)
		preserved := sameTerminal(beforeTrace, expectedBefore) && sameTerminal(afterTrace, expectedAfter)
		if corpus.Class != "counterexample" {
			preserved = preserved && unchanged
		}
		result.Evidence = append(result.Evidence, CorpusEvidence{
			InputID: corpus.ID, Class: corpus.Class, Before: beforeTrace, After: afterTrace,
			ExpectedBefore: expectedBefore, ExpectedAfter: expectedAfter, Unchanged: unchanged, Preserved: preserved,
		})
		result.PairedIndicator = append(result.PairedIndicator, corpus.ID+"="+beforeTrace.Decision+"->"+afterTrace.Decision+":"+indicatorValue(preserved))
		if corpus.Class != "counterexample" && !preserved {
			result.CorpusPreserved = false
		}
	}
	if !result.CounterexampleRemoved && !hasCounterexampleHidden(before) && after.Terminal.CounterexampleVisible {
		result.KnownContradiction = true
		result.Contradiction = "candidate did not remove the causal counterexample"
	}
	if hasCounterexampleHidden(before) && !result.CounterexampleRemoved {
		result.KnownContradiction = true
		result.Contradiction = "candidate hides the causal counterexample without proving its removal"
	}
	if !result.CorpusPreserved {
		result.KnownContradiction = true
		result.Contradiction = "candidate changes a normal or regression corpus behavior"
	}
	return result
}

func exactEvaluationIdentity(meta MetaContract, c Counterexample, before, after SemanticIR) bool {
	return c.ContractDigest != "" && c.ContractDigest == meta.ContractDigest && c.EvaluatorID == meta.Evaluator.ID && c.EvaluatorDigest == meta.Evaluator.Digest &&
		before.Scenario == c.Scenario && after.Scenario == c.Scenario && before.OriginSourceDigest == c.OriginSourceDigest && after.OriginSourceDigest == c.OriginSourceDigest &&
		before.ContractDigest == c.ContractDigest && after.ContractDigest == c.ContractDigest && before.ToolchainDigest == c.ToolchainDigest && after.ToolchainDigest == c.ToolchainDigest &&
		before.CandidateSpaceID == meta.CandidateSpace.ID && after.CandidateSpaceID == meta.CandidateSpace.ID && before.CandidateSpaceDigest == meta.CandidateSpace.Digest && after.CandidateSpaceDigest == meta.CandidateSpace.Digest &&
		before.RuleID == meta.Rule.ID && after.RuleID == meta.Rule.ID && before.RuleDigest == meta.Rule.Digest && after.RuleDigest == meta.Rule.Digest &&
		before.EvaluatorID == c.EvaluatorID && after.EvaluatorID == c.EvaluatorID && before.EvaluatorDigest == c.EvaluatorDigest && after.EvaluatorDigest == c.EvaluatorDigest
}

func evaluateCorpusInput(ir SemanticIR, c Counterexample, corpus CorpusCase, candidate bool) TerminalTrace {
	if corpus.Class == "counterexample" {
		if candidate {
			if !hasResolutionProof(ir, c) {
				return TerminalTrace{Decision: DecisionUnknown, Reason: "causal repair proof missing", ReasonDigest: DigestString("causal repair proof missing"), EffectTrace: []string{}, CounterexampleVisible: true}
			}
			return ir.Terminal.Normalized()
		}
		return c.Baseline.Normalized()
	}
	behavior := ""
	reason := "corpus " + corpus.Input
	effects := []string{}
	for _, node := range ir.Nodes {
		attrs := attributes(node)
		if value := attrs["behavior_"+corpus.Input]; value != "" {
			behavior = value
		}
		if value := attrs["reason_"+corpus.Input]; value != "" {
			reason = value
		}
		if value := attrs["effects_"+corpus.Input]; value != "" {
			effects = splitCSV(value)
		}
	}
	if behavior == "" {
		return ir.Terminal.Normalized()
	}
	return TerminalTrace{Decision: behavior, Reason: reason, ReasonDigest: DigestString(reason), EffectTrace: effects, CounterexampleVisible: false}
}

func hasResolutionProof(ir SemanticIR, c Counterexample) bool {
	proofID := c.CausalInput.ID + "::proof"
	for _, node := range ir.Nodes {
		attrs := attributes(node)
		if node.ID == proofID && node.Kind == "proof" && attrs["causal_input"] == c.CausalInput.ID && attrs["decision"] == DecisionClosed {
			return true
		}
	}
	return false
}

func sameTerminal(left, right TerminalTrace) bool {
	left = left.Normalized()
	right = right.Normalized()
	return left.Decision == right.Decision && left.Reason == right.Reason && left.ReasonDigest == right.ReasonDigest && sameStrings(left.EffectTrace, right.EffectTrace)
}

func indicatorValue(value bool) string {
	if value {
		return "PRESERVED"
	}
	return "CHANGED"
}

func hasCounterexampleHidden(ir SemanticIR) bool {
	for _, node := range ir.Nodes {
		if attributes(node)["hide_counterexample"] == "true" {
			return true
		}
	}
	return false
}

func validateExternalUtility(c Counterexample) *UnknownRecord {
	if c.ExternalUtility.Name == "" {
		return nil
	}
	if c.ExternalUtility.Release == "" || !validDigest(c.ExternalUtility.Digest) || !c.ExternalUtility.Available {
		return unknown("input", "resolve_external_utility", "external utility evidence is unavailable or not digest-pinned", "EXTERNAL_UTILITY_UNKNOWN", "provide_immutable_utility_evidence", "external-utility")
	}
	return nil
}

func declaredIdentityMissing(c Counterexample) []string {
	fields := c.IdentityFields()
	missing := make([]string, 0, len(fields))
	for _, name := range []string{"candidate-space-id", "candidate-space-digest", "rule-id", "rule-digest", "evaluator-id", "evaluator-digest"} {
		if strings.TrimSpace(fields[name]) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}
