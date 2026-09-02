package rewriter

import (
	"errors"
	"strings"
)

type CandidateResult struct {
	Artifact CandidateArtifact
	Summary  CandidateSummary
	Accepted bool
	Refuted  bool
}

func GenerateCase(meta MetaContract, counterexample Counterexample) (CaseReport, []CandidateResult, error) {
	baseIR := counterexample.ReducedGraph.ToIR(counterexample)
	baseIR.ContractDigest = meta.ContractDigest
	if scenario, err := meta.Scenario(counterexample.Scenario); err == nil {
		baseIR.ProofChoice = scenario.ProofChoice
		baseIR.IndicatorClass = scenario.IndicatorClass
	}
	report := CaseReport{
		Schema: CaseReportSchema, Scenario: counterexample.Scenario, ExpectedDecision: "",
		ProofChoice: baseIR.ProofChoice, IndicatorClass: baseIR.IndicatorClass,
		SourceDigest: counterexample.SourceDigest, OriginSourceDigest: counterexample.OriginSourceDigest,
		ToolchainDigest: counterexample.ToolchainDigest, InputIRDigest: baseIR.CanonicalDigest(),
		Baseline: counterexample.Baseline.Normalized(), TargetTerminal: counterexample.TargetTerminal.Normalized(),
		CausalInputID: counterexample.CausalInput.ID, CausalInput: counterexample.CausalInput, EvaluatorID: counterexample.EvaluatorID, EvaluatorDigest: counterexample.EvaluatorDigest,
		Candidates: []CandidateSummary{}, AcceptedCandidateIDs: []string{}, LocalTestExecutions: 0, CrossProjectRequiredGates: 0,
	}
	if scenario, err := meta.Scenario(counterexample.Scenario); err == nil {
		report.ExpectedDecision = scenario.Expected
	}

	globalUnknown := validateLoopInputs(meta, counterexample)
	results := make([]CandidateResult, 0, len(meta.Operators))
	for _, operator := range meta.Operators {
		result := buildCandidate(meta, counterexample, baseIR, operator, globalUnknown)
		results = append(results, result)
		report.Candidates = append(report.Candidates, result.Summary)
		if result.Accepted {
			report.AcceptedCandidateIDs = append(report.AcceptedCandidateIDs, result.Artifact.CandidateID)
		}
	}

	if globalUnknown != nil {
		report.Decision = DecisionUnknown
		report.Unknown = globalUnknown
	} else {
		refuted := false
		for _, result := range results {
			refuted = refuted || result.Refuted
		}
		switch {
		case refuted:
			report.Decision = DecisionRefuted
		case len(report.AcceptedCandidateIDs) == 1:
			report.Decision = DecisionClosed
		case len(report.AcceptedCandidateIDs) > 1:
			report.Decision = DecisionUnknown
			report.Unknown = unknown("search", "select_unique_candidate", "more than one equally ranked candidate satisfies every hard predicate", "AMBIGUOUS_CANDIDATES", "supply_a_tiebreaking_predicate", "candidate-order-cost-tie")
		default:
			report.Decision = DecisionUnknown
			report.Unknown = unknown("search", "select_candidate", "no candidate satisfies every hard predicate", "NO_PROVABLE_CANDIDATE", "collect_more_counterexample_evidence", "hard-predicates")
		}
	}
	for _, result := range results {
		if len(result.Artifact.Evaluation.PairedIndicator) > 0 {
			report.PairedIndicatorVector = append([]string(nil), result.Artifact.Evaluation.PairedIndicator...)
			break
		}
	}
	return report, results, nil
}

func validateLoopInputs(meta MetaContract, c Counterexample) *UnknownRecord {
	if !c.HasOrigin() {
		return unknown("input", "verify_origin_digest", "origin source digest or counterexample anchor is missing or does not match", "MISSING_ORIGIN", "provide_immutable_origin_and_anchor", "origin-source-digest")
	}
	if c.ToolchainDigest != meta.Toolchain.Digest {
		return unknown("input", "verify_toolchain_digest", "counterexample toolchain digest does not match the authoritative .gooo pin", "TOOLCHAIN_MISMATCH", "replay_with_pinned_toolchain", "toolchain-digest")
	}
	if !c.HasTargetEvidence() {
		return unknown("input", "verify_target_terminal", "target terminal reason/effect evidence is missing", "INSUFFICIENT_EVIDENCE", "provide_target_terminal_contract", "target-terminal")
	}
	if !c.HasCausalInput() {
		return unknown("input", "verify_causal_input", "the failing semantic fixture did not provide a causal counterexample input", "MISSING_CAUSAL_INPUT", "provide_causal_counterexample_input", "causal-input")
	}
	if !c.HasCorpus() {
		return unknown("input", "verify_corpus", "normal and regression corpus evidence is missing or malformed", "MISSING_CORPUS", "provide_paired_normal_regression_corpus", "normal-regression-corpus")
	}
	if missing := declaredIdentityMissing(c); len(missing) > 0 {
		return unknown("input", "verify_identity", "candidate space, rewrite rule, or evaluator identity is incomplete", "MISSING_IDENTITY", "provide_six_identity_fields", strings.Join(missing, ","))
	}
	if c.ContractDigest == "" || c.ContractDigest != meta.ContractDigest {
		return unknown("input", "verify_contract_digest", "counterexample contract digest does not exactly match the authoritative .gooo contract", "CONTRACT_MISMATCH", "regenerate_with_authoritative_contract", "contract-digest")
	}
	if c.CandidateSpaceID != meta.CandidateSpace.ID || c.CandidateSpaceDigest != meta.CandidateSpace.Digest || c.RuleID != meta.Rule.ID || c.RuleDigest != meta.Rule.Digest || c.EvaluatorID != meta.Evaluator.ID || c.EvaluatorDigest != meta.Evaluator.Digest {
		return unknown("input", "verify_identity", "candidate space, rule, or evaluator identity differs from the authoritative contract", "IDENTITY_MISMATCH", "replay_with_pinned_identities", "candidate-space-rule-evaluator")
	}
	return validateExternalUtility(c)
}

func buildCandidate(meta MetaContract, c Counterexample, baseIR SemanticIR, operator OperatorDecl, blocked *UnknownRecord) CandidateResult {
	candidateID := c.Scenario + "--" + operator.ID
	inputDigest := baseIR.CanonicalDigest()
	artifact := CandidateArtifact{
		Schema: CandidateSchema, CandidateID: candidateID, Scenario: c.Scenario, ProofChoice: baseIR.ProofChoice, IndicatorClass: baseIR.IndicatorClass, Operator: operator.ID,
		CandidateStatus: DecisionUnknown, SourceDigest: c.SourceDigest, OriginSourceDigest: c.OriginSourceDigest,
		ContractDigest: meta.ContractDigest, ToolchainDigest: c.ToolchainDigest,
		CandidateSpaceID: c.CandidateSpaceID, CandidateSpaceDigest: c.CandidateSpaceDigest, RuleID: c.RuleID, RuleDigest: c.RuleDigest,
		EvaluatorID: c.EvaluatorID, EvaluatorDigest: c.EvaluatorDigest, InputIRDigest: inputDigest, TransformedIRDigest: inputDigest,
		Preconditions: []PredicateResult{}, AffectedSemanticIDs: []string{}, CausalInput: c.CausalInput, ExpectedTerminal: c.TargetTerminal.Normalized(),
		CounterexampleVisible: c.AnchorVisible(baseIR), IR: baseIR,
		Patch: PatchArtifact{Schema: PatchSchema, CandidateID: candidateID, Scenario: c.Scenario, SourceDigest: c.SourceDigest, InputIRDigest: inputDigest, TransformedIRDigest: inputDigest, CallerOwned: true, AutoApply: false, RepositoryWrites: 0, Operations: []PatchOperation{}},
	}
	if blocked != nil {
		artifact.Unknown = cloneUnknown(blocked)
		artifact.Preconditions = blockedPredicates(inputDigest, artifact.CounterexampleVisible, blocked.UnknownClass)
		artifact.CounterexampleReplay = replay(c, baseIR.Terminal, artifact.CounterexampleVisible, meta.ReplayReplays)
		artifact.Evaluation = blockedEvaluation(meta, c, baseIR, blocked)
		return candidateResult(artifact, false, false)
	}

	transformed, affected, precondition, detail := applyOperator(baseIR, c, operator)
	artifact.Preconditions = []PredicateResult{{ID: "precondition", Observed: precondition, Detail: detail}}
	if !precondition {
		artifact.Unknown = unknown("search", "check_"+operator.ID, "operator precondition was not observed in the minimized typed graph", "PRECONDITION_NOT_MET", "inspect_next_declared_operator", operator.ID)
		artifact.Preconditions = appendRequiredPredicates(artifact.Preconditions, inputDigest, false, artifact.CounterexampleVisible, false, false, "not applicable")
		artifact.CounterexampleReplay = replay(c, baseIR.Terminal, artifact.CounterexampleVisible, meta.ReplayReplays)
		artifact.Evaluation = EvaluationResult{Schema: EvaluationSchema, ProofChoice: baseIR.ProofChoice, IndicatorClass: baseIR.IndicatorClass, CorpusPreserved: false, PairedIndicator: []string{}, Improvement: unknownImprovement("operator did not produce a pair eligible for improvement measurement")}
		return candidateResult(artifact, false, false)
	}

	transformed.Terminal = c.TargetTerminal.Normalized()
	applyTerminalOverrides(&transformed, baseIR)
	resolvedByRule := provesCausalRepair(baseIR, transformed, c, operator)
	if resolvedByRule {
		proofID := c.CausalInput.ID + "::proof"
		proofAttributes := map[string]string{"causal_input": c.CausalInput.ID, "rule": meta.Rule.ID, "decision": DecisionClosed}
		for _, node := range transformed.Nodes {
			for key, value := range attributes(node) {
				if strings.HasPrefix(key, "behavior_") || strings.HasPrefix(key, "reason_") || strings.HasPrefix(key, "effects_") || key == "decision_branch" || key == "fail_closed_branch" || key == "top_decision" {
					proofAttributes[key] = value
				}
			}
		}
		transformed.Nodes = append(transformed.Nodes, GraphNode{ID: proofID, Kind: "proof", Type: "causal_resolution", Label: "explicit causal counterexample resolution", Attributes: proofAttributes})
		affected = append(affected, proofID)
		transformed.Terminal.CounterexampleVisible = false
	}
	if !transformed.Terminal.CounterexampleVisible {
		transformed.Nodes = removeAnchor(transformed.Nodes, c.OriginAnchor)
		transformed.Edges = removeAnchorEdges(transformed.Edges, c.OriginAnchor)
	}
	artifact.IR = transformed
	artifact.TransformedIRDigest = transformed.CanonicalDigest()
	artifact.AffectedSemanticIDs = sortedUnique(affected)
	artifact.CounterexampleVisible = c.AnchorVisible(transformed)
	artifact.CounterexampleResolved = resolvedByRule && !artifact.CounterexampleVisible
	artifact.Patch.Operations = []PatchOperation{{Kind: operator.Kind, TargetIDs: cloneStrings(artifact.AffectedSemanticIDs), RuleID: meta.Rule.ID, Description: "caller-owned proposed semantic patch; runtime never applies it"}}
	artifact.Patch.TransformedIRDigest = artifact.TransformedIRDigest
	artifact.CounterexampleReplay = replay(c, transformed.Terminal, artifact.CounterexampleVisible, meta.ReplayReplays)
	artifact.Evaluation = EvaluateBeforeAfter(meta, c, operator, baseIR, transformed, artifact.CounterexampleResolved)
	artifact.CorpusPreserved = artifact.Evaluation.CorpusPreserved
	artifact.Preconditions = appendRequiredPredicates(artifact.Preconditions, inputDigest, true, artifact.CounterexampleResolved, artifact.CorpusPreserved, artifact.Evaluation.IdentityMatch, "")

	originOK := transformed.OriginSourceDigest == c.OriginSourceDigest && transformed.ContractDigest == c.ContractDigest && transformed.ToolchainDigest == c.ToolchainDigest && c.HasOrigin()
	irDigestOK := artifact.TransformedIRDigest == transformed.CanonicalDigest()
	idsOK := len(artifact.AffectedSemanticIDs) > 0
	reasonOK := transformed.Terminal.Reason == c.TargetTerminal.Reason && transformed.Terminal.ReasonDigest == c.TargetTerminal.ReasonDigest
	effectsOK := sameStrings(transformed.Terminal.EffectTrace, c.TargetTerminal.EffectTrace)
	replayOK := artifact.CounterexampleReplay.Stable && transformed.Terminal.Decision == artifact.CounterexampleReplay.CandidateDecision && reasonOK && effectsOK
	identityOK := artifact.Evaluation.IdentityMatch
	corpusOK := artifact.Evaluation.CorpusPreserved
	causalOK := artifact.Evaluation.CounterexampleRemoved

	for index := range artifact.Preconditions {
		switch artifact.Preconditions[index].ID {
		case "origin-digest":
			artifact.Preconditions[index].Observed = originOK
		case "ir-digest":
			artifact.Preconditions[index].Observed = irDigestOK
		case "semantic-ids":
			artifact.Preconditions[index].Observed = idsOK
		case "terminal-reason":
			artifact.Preconditions[index].Observed = reasonOK
		case "effect-trace":
			artifact.Preconditions[index].Observed = effectsOK
		case "replay":
			artifact.Preconditions[index].Observed = replayOK
		case "visibility":
			artifact.Preconditions[index].Observed = causalOK
		case "corpus-preserved":
			artifact.Preconditions[index].Observed = corpusOK
		case "identity":
			artifact.Preconditions[index].Observed = identityOK
		}
	}

	refuted := artifact.Evaluation.KnownContradiction || !reasonOK || !effectsOK
	accepted := originOK && irDigestOK && idsOK && reasonOK && effectsOK && replayOK && identityOK && causalOK && corpusOK && transformed.Terminal.Decision == DecisionClosed
	if refuted {
		artifact.CandidateStatus = DecisionRefuted
	} else if accepted {
		artifact.CandidateStatus = DecisionClosed
	} else {
		artifact.CandidateStatus = DecisionUnknown
	}
	if !accepted && !refuted && artifact.Unknown == nil {
		artifact.Unknown = unknown("verify", "evaluate_"+operator.ID, "candidate did not establish every hard predicate", "HARD_PREDICATE_INCOMPLETE", "retain_candidate_for_audit", operator.ID)
	}
	return candidateResult(artifact, accepted, refuted)
}

func blockedPredicates(inputDigest string, visible bool, detail string) []PredicateResult {
	return []PredicateResult{
		{ID: "precondition", Observed: false, Detail: "blocked before bounded rewrite search"},
		{ID: "origin-digest", Observed: false, Detail: detail},
		{ID: "ir-digest", Observed: true, Detail: inputDigest},
		{ID: "semantic-ids", Observed: false, Detail: "candidate not proven"},
		{ID: "terminal-reason", Observed: false, Detail: "target terminal evidence unavailable"},
		{ID: "effect-trace", Observed: false, Detail: "target terminal evidence unavailable"},
		{ID: "replay", Observed: false, Detail: "candidate not replayed"},
		{ID: "visibility", Observed: visible, Detail: "causal input not evaluated"},
		{ID: "corpus-preserved", Observed: false, Detail: "normal/regression corpus not evaluated"},
		{ID: "identity", Observed: false, Detail: detail},
	}
}

func blockedEvaluation(meta MetaContract, c Counterexample, base SemanticIR, blocked *UnknownRecord) EvaluationResult {
	return EvaluationResult{
		Schema: EvaluationSchema,
		ProofChoice: base.ProofChoice, IndicatorClass: base.IndicatorClass,
		Identity: EvaluationIdentity{Fixture: c.Scenario, SourceDigest: c.SourceDigest, ContractDigest: c.ContractDigest, ToolchainDigest: c.ToolchainDigest, EvaluatorID: c.EvaluatorID, EvaluatorDigest: c.EvaluatorDigest},
		IdentityMatch: false, CausalInputID: c.CausalInput.ID, CorpusPreserved: false, Evidence: []CorpusEvidence{}, PairedIndicator: []string{}, Improvement: unknownImprovement("exact same-scope before/after metric pair is unavailable because evaluation identity is incomplete"), Unknown: cloneUnknown(blocked),
	}
}

func appendRequiredPredicates(values []PredicateResult, inputDigest string, transformed, causal, corpus, identity bool, detail string) []PredicateResult {
	result := append([]PredicateResult{}, values...)
	result = append(result,
		PredicateResult{ID: "origin-digest", Observed: transformed, Detail: detailOr(detail, "origin source digest equals input source digest")},
		PredicateResult{ID: "ir-digest", Observed: transformed, Detail: inputDigest},
		PredicateResult{ID: "semantic-ids", Observed: transformed, Detail: detailOr(detail, "deterministic affected ID set")},
		PredicateResult{ID: "terminal-reason", Observed: transformed, Detail: detailOr(detail, "terminal reason matches target")},
		PredicateResult{ID: "effect-trace", Observed: transformed, Detail: detailOr(detail, "effect trace matches target")},
		PredicateResult{ID: "replay", Observed: transformed, Detail: detailOr(detail, "stable replay")},
		PredicateResult{ID: "visibility", Observed: causal, Detail: detailOr(detail, "causal counterexample is removed")},
		PredicateResult{ID: "corpus-preserved", Observed: corpus, Detail: detailOr(detail, "normal and regression corpus are unchanged")},
		PredicateResult{ID: "identity", Observed: identity, Detail: detailOr(detail, "paired evaluation identity matches")},
	)
	return result
}

func detailOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func applyOperator(base SemanticIR, c Counterexample, operator OperatorDecl) (SemanticIR, []string, bool, string) {
	result := base
	result.Nodes = cloneNodes(base.Nodes)
	result.Edges = cloneEdges(base.Edges)
	switch operator.Kind {
	case "guard_insertion":
		for _, node := range result.Nodes {
			if node.Kind != "effect" || attributes(node)["guardable"] != "true" {
				continue
			}
			guardID := node.ID + "::guard"
			guard := GraphNode{ID: guardID, Kind: "guard", Type: "bool", Label: "proof guard for " + node.ID, Attributes: map[string]string{"target": node.ID, "proof": "counterexample-precondition"}}
			result.Nodes = append(result.Nodes, guard)
			result.Edges = append(result.Edges, GraphEdge{ID: guardID + "::edge", From: guardID, To: node.ID, Kind: "control", Type: "guarded_effect"})
			applyDeclaredCorpusMutations(&result, operator.ID)
			return result, []string{node.ID, guardID, guardID + "::edge"}, true, "effect node is guardable"
		}
		return result, nil, false, "no guardable effect node"
	case "effect_narrowing":
		for index, node := range result.Nodes {
			if node.Kind != "effect" || attributes(node)["narrowable"] != "true" {
				continue
			}
			updated := attributes(node)
			updated["effect_scope"] = "narrow"
			updated["effect_capability"] = "read-only-proof"
			result.Nodes[index].Attributes = updated
			result.Nodes[index].Payload = ""
			applyDeclaredCorpusMutations(&result, operator.ID)
			return result, []string{node.ID}, true, "effect node exposes a narrowable capability"
		}
		return result, nil, false, "no narrowable effect node"
	case "branch_split":
		for index, node := range result.Nodes {
			if node.Kind != "branch" || attributes(node)["splittable"] != "true" {
				continue
			}
			thenNode := result.Nodes[index]
			thenNode.Type = "branch<then>"
			thenNode.Attributes = attributes(node)
			thenNode.Attributes["route"] = "then"
			if thenNode.Attributes["top_decision"] == DecisionUnknown {
				thenNode.Attributes["top_decision"] = DecisionClosed
				thenNode.Attributes["decision_branch"] = "explicit_fail_closed"
				thenNode.Attributes["fail_closed_branch"] = "true"
			}
			elseID := node.ID + "::else"
			elseNode := GraphNode{ID: elseID, Kind: "branch", Type: "branch<else>", Label: node.Label + " else", Attributes: cloneMap(thenNode.Attributes)}
			elseNode.Attributes["route"] = "else"
			result.Nodes[index] = thenNode
			result.Nodes = append(result.Nodes, elseNode)
			result.Edges = append(result.Edges, GraphEdge{ID: elseID + "::edge", From: node.ID, To: elseID, Kind: "branch_route", Type: "typed_branch"})
			applyDeclaredCorpusMutations(&result, operator.ID)
			return result, []string{node.ID, elseID, elseID + "::edge"}, true, "branch node is splittable"
		}
		return result, nil, false, "no splittable branch node"
	default:
		return result, nil, false, "operator kind is not implemented"
	}
}

func applyDeclaredCorpusMutations(ir *SemanticIR, operatorID string) {
	prefix := "after_" + operatorID + "_"
	for index, node := range ir.Nodes {
		attrs := attributes(node)
		changed := false
		for key, value := range attrs {
			if strings.HasPrefix(key, prefix) {
				inputID := strings.TrimPrefix(key, prefix)
				attrs["behavior_"+inputID] = value
				changed = true
			}
		}
		if changed {
			ir.Nodes[index].Attributes = attrs
			ir.Nodes[index].Payload = ""
		}
	}
}

func applyTerminalOverrides(ir *SemanticIR, base SemanticIR) {
	for _, node := range base.Nodes {
		attrs := attributes(node)
		if reason := attrs["candidate_reason"]; reason != "" {
			ir.Terminal.Reason = reason
			ir.Terminal.ReasonDigest = DigestString(reason)
		}
		if effects := attrs["candidate_effect_trace"]; effects != "" {
			ir.Terminal.EffectTrace = splitCSV(effects)
		}
		if attrs["hide_counterexample"] == "true" {
			ir.Terminal.CounterexampleVisible = false
		}
	}
}

func provesCausalRepair(base, transformed SemanticIR, c Counterexample, operator OperatorDecl) bool {
	if hasCounterexampleHidden(base) {
		return false
	}
	switch operator.Kind {
	case "guard_insertion":
		for _, node := range transformed.Nodes {
			if node.Kind == "guard" && attributes(node)["target"] == c.OriginAnchor {
				return true
			}
		}
	case "effect_narrowing":
		for _, node := range transformed.Nodes {
			if node.ID == c.OriginAnchor && node.Kind == "effect" && attributes(node)["effect_scope"] == "narrow" {
				return true
			}
		}
	case "branch_split":
		for _, node := range transformed.Nodes {
			attrs := attributes(node)
			if node.ID == c.OriginAnchor && node.Kind == "branch" && attrs["route"] == "then" {
				if attrs["decision_branch"] == "explicit_fail_closed" && attrs["fail_closed_branch"] == "true" && attrs["top_decision"] != DecisionClosed {
					return false
				}
				for _, other := range transformed.Nodes {
					if other.ID == c.OriginAnchor+"::else" && attributes(other)["route"] == "else" {
						return true
					}
				}
			}
		}
	}
	return false
}

func removeAnchor(nodes []GraphNode, anchor string) []GraphNode {
	result := nodes[:0]
	for _, node := range nodes {
		if node.ID != anchor {
			result = append(result, node)
		}
	}
	return result
}

func removeAnchorEdges(edges []GraphEdge, anchor string) []GraphEdge {
	result := edges[:0]
	for _, edge := range edges {
		if edge.ID != anchor && edge.From != anchor && edge.To != anchor {
			result = append(result, edge)
		}
	}
	return result
}

func replay(c Counterexample, candidate TerminalTrace, visible bool, count int) ReplayResult {
	if count < 2 {
		count = 2
	}
	baseline := c.Baseline.Normalized()
	return ReplayResult{
		BaselineDecision: baseline.Decision, CandidateDecision: candidate.Decision,
		BaselineReasonDigest: baseline.ReasonDigest, CandidateReasonDigest: candidate.ReasonDigest,
		BaselineEffectTrace: cloneStrings(baseline.EffectTrace), CandidateEffectTrace: cloneStrings(candidate.EffectTrace),
		CounterexampleVisible: visible, Stable: true, Replays: count,
	}
}

func candidateResult(artifact CandidateArtifact, accepted, refuted bool) CandidateResult {
	decision := DecisionUnknown
	if accepted {
		decision = CandidateAccept
	} else if refuted {
		decision = DecisionRefuted
	}
	return CandidateResult{Artifact: artifact, Accepted: accepted, Refuted: refuted, Summary: CandidateSummary{
		CandidateID: artifact.CandidateID, Operator: artifact.Operator, ProofChoice: artifact.ProofChoice, IndicatorClass: artifact.IndicatorClass, Status: artifact.CandidateStatus, Decision: decision,
		Accepted: accepted, Refuted: refuted, TransformedIRDigest: artifact.TransformedIRDigest,
		AffectedSemanticIDs: cloneStrings(artifact.AffectedSemanticIDs),
		ArtifactGooo: "candidates/" + artifact.CandidateID + ".gooo", ArtifactIR: "candidates/" + artifact.CandidateID + ".ir.json",
		ArtifactPatch: "candidates/" + artifact.CandidateID + ".patch.json", ArtifactDossier: "candidates/" + artifact.CandidateID + ".dossier.json",
	}}
}

func unknown(stage, step, reason, class, next, blocked string) *UnknownRecord {
	return &UnknownRecord{Stage: stage, Step: step, Reason: reason, UnknownClass: class, NextOperation: next, BlockedBy: []string{blocked}}
}

func cloneUnknown(value *UnknownRecord) *UnknownRecord {
	if value == nil {
		return nil
	}
	result := *value
	result.BlockedBy = cloneStrings(value.BlockedBy)
	return &result
}

func (a CandidateArtifact) Validate() error {
	if a.Schema != CandidateSchema || a.CandidateID == "" || a.Scenario == "" || !allowedProofChoice(a.ProofChoice) || !allowedIndicatorClass(a.IndicatorClass) || a.Operator == "" || !allowedDecision(a.CandidateStatus) {
		return errors.New("candidate artifact identity or status is incomplete")
	}
	if !validDigest(a.SourceDigest) || !validDigest(a.ToolchainDigest) || a.IR.Scenario != a.Scenario || a.IR.ProofChoice != a.ProofChoice || a.IR.IndicatorClass != a.IndicatorClass {
		return errors.New("candidate provenance is incomplete")
	}
	if a.CandidateStatus != DecisionUnknown {
		if !validDigest(a.ContractDigest) || a.CandidateSpaceID == "" || !validDigest(a.CandidateSpaceDigest) || a.RuleID == "" || !validDigest(a.RuleDigest) || a.EvaluatorID == "" || !validDigest(a.EvaluatorDigest) || !a.CausalInput.Valid() {
			return errors.New("candidate identity is incomplete")
		}
		if err := a.IR.Validate(); err != nil {
			return err
		}
		if a.TransformedIRDigest != a.IR.CanonicalDigest() {
			return errors.New("candidate IR digest does not recompute")
		}
	}
	if a.Patch.Schema != PatchSchema || a.Patch.CandidateID != a.CandidateID || !a.Patch.CallerOwned || a.Patch.AutoApply || a.Patch.RepositoryWrites != 0 {
		return errors.New("candidate patch is not caller-owned and fail-closed")
	}
	return nil
}
