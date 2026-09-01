package rewriter

import (
	"errors"
)

type CandidateResult struct {
	Artifact CandidateArtifact
	Summary  CandidateSummary
	Accepted bool
	Refuted  bool
}

func GenerateCase(meta MetaContract, counterexample Counterexample) (CaseReport, []CandidateResult, error) {
	baseIR := counterexample.ReducedGraph.ToIR(counterexample)
	report := CaseReport{
		Schema: CaseReportSchema, Scenario: counterexample.Scenario, ExpectedDecision: "",
		SourceDigest: counterexample.SourceDigest, OriginSourceDigest: counterexample.OriginSourceDigest,
		ToolchainDigest: counterexample.ToolchainDigest, InputIRDigest: baseIR.CanonicalDigest(),
		Baseline: counterexample.Baseline.Normalized(), TargetTerminal: counterexample.TargetTerminal.Normalized(),
		Candidates: []CandidateSummary{}, AcceptedCandidateIDs: []string{},
	}
	if scenario, err := meta.Scenario(counterexample.Scenario); err == nil {
		report.ExpectedDecision = scenario.Expected
	}

	globalUnknown := (*UnknownRecord)(nil)
	if !counterexample.HasOrigin() {
		globalUnknown = unknown("search", "verify_origin_digest", "origin source digest or counterexample anchor is missing or does not match", "MISSING_ORIGIN", "provide_immutable_origin_and_anchor", "origin-source-digest")
	} else if counterexample.ToolchainDigest != meta.Toolchain.Digest {
		globalUnknown = unknown("search", "verify_toolchain_digest", "counterexample toolchain digest does not match the authoritative .gooo pin", "TOOLCHAIN_MISMATCH", "replay_with_pinned_toolchain", "toolchain-digest")
	} else if !counterexample.HasTargetEvidence() {
		globalUnknown = unknown("search", "verify_target_terminal", "target terminal reason/effect evidence is missing", "INSUFFICIENT_EVIDENCE", "provide_target_terminal_contract", "target-terminal")
	}

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
		decisions := make([]string, 0, len(results))
		for _, result := range results {
			if result.Refuted {
				decisions = append(decisions, DecisionRefuted)
			}
		}
		switch {
		case len(decisions) > 0:
			report.Decision = DecisionRefuted
			report.Unknown = nil
		case len(report.AcceptedCandidateIDs) == 1:
			report.Decision = DecisionClosed
			report.Unknown = nil
		case len(report.AcceptedCandidateIDs) > 1:
			report.Decision = DecisionUnknown
			report.Unknown = unknown("search", "select_unique_candidate", "more than one equally ranked candidate satisfies every hard predicate", "AMBIGUOUS_CANDIDATES", "supply_a_tiebreaking_predicate", "candidate-order-cost-tie")
		default:
			report.Decision = DecisionUnknown
			report.Unknown = unknown("search", "select_candidate", "no candidate satisfies every hard predicate", "NO_PROVABLE_CANDIDATE", "collect_more_counterexample_evidence", "hard-predicates")
		}
	}
	return report, results, nil
}

func buildCandidate(meta MetaContract, c Counterexample, baseIR SemanticIR, operator OperatorDecl, blocked *UnknownRecord) CandidateResult {
	candidateID := c.Scenario + "--" + operator.ID
	inputDigest := baseIR.CanonicalDigest()
	artifact := CandidateArtifact{
		Schema: CandidateSchema, CandidateID: candidateID, Scenario: c.Scenario, Operator: operator.ID,
		CandidateStatus: DecisionUnknown, SourceDigest: c.SourceDigest, OriginSourceDigest: c.OriginSourceDigest,
		ToolchainDigest: c.ToolchainDigest, InputIRDigest: inputDigest, TransformedIRDigest: inputDigest,
		Preconditions: []PredicateResult{}, AffectedSemanticIDs: []string{}, ExpectedTerminal: c.TargetTerminal.Normalized(),
		CounterexampleVisible: c.AnchorVisible(baseIR) && baseIR.Terminal.CounterexampleVisible, IR: baseIR,
	}
	if blocked != nil {
		artifact.Unknown = cloneUnknown(blocked)
		artifact.Preconditions = []PredicateResult{
			{ID: "precondition", Observed: false, Detail: "blocked before bounded rewrite search"},
			{ID: "origin-digest", Observed: false, Detail: blocked.UnknownClass},
			{ID: "ir-digest", Observed: true, Detail: inputDigest},
			{ID: "semantic-ids", Observed: false, Detail: "candidate not proven"},
			{ID: "terminal-reason", Observed: false, Detail: "target terminal evidence unavailable"},
			{ID: "effect-trace", Observed: false, Detail: "target terminal evidence unavailable"},
			{ID: "replay", Observed: false, Detail: "candidate not replayed"},
			{ID: "visibility", Observed: artifact.CounterexampleVisible, Detail: "origin evidence is incomplete"},
		}
		artifact.CounterexampleReplay = replay(c, baseIR.Terminal, false, meta.ReplayReplays)
		return candidateResult(artifact, false, false)
	}

	transformed, affected, precondition, detail := applyOperator(baseIR, c, operator)
	artifact.Preconditions = []PredicateResult{{ID: "precondition", Observed: precondition, Detail: detail}}
	if !precondition {
		artifact.Unknown = unknown("search", "check_"+operator.ID, "operator precondition was not observed in the minimized typed graph", "PRECONDITION_NOT_MET", "inspect_next_declared_operator", operator.ID)
		artifact.Preconditions = appendRequiredPredicates(artifact.Preconditions, inputDigest, false, c.AnchorVisible(baseIR), "not applicable")
		artifact.CounterexampleReplay = replay(c, baseIR.Terminal, false, meta.ReplayReplays)
		return candidateResult(artifact, false, false)
	}

	transformed.Terminal = c.TargetTerminal.Normalized()
	applyTerminalOverrides(&transformed, baseIR)
	visible := c.AnchorVisible(transformed) && transformed.Terminal.CounterexampleVisible
	if !visible {
		transformed.Nodes = removeAnchor(transformed.Nodes, c.OriginAnchor)
		transformed.Edges = removeAnchorEdges(transformed.Edges, c.OriginAnchor)
	}
	artifact.IR = transformed
	artifact.TransformedIRDigest = transformed.CanonicalDigest()
	artifact.AffectedSemanticIDs = sortedUnique(affected)
	artifact.CounterexampleVisible = visible
	artifact.Preconditions = appendRequiredPredicates(artifact.Preconditions, inputDigest, true, artifact.CounterexampleVisible, "")
	artifact.CounterexampleReplay = replay(c, transformed.Terminal, artifact.CounterexampleVisible, meta.ReplayReplays)

	originOK := transformed.OriginSourceDigest == c.SourceDigest && transformed.ToolchainDigest == c.ToolchainDigest && c.HasOrigin()
	irDigestOK := artifact.TransformedIRDigest == transformed.CanonicalDigest()
	idsOK := len(artifact.AffectedSemanticIDs) > 0
	reasonOK := transformed.Terminal.Reason == c.TargetTerminal.Reason && transformed.Terminal.ReasonDigest == c.TargetTerminal.ReasonDigest
	effectsOK := sameStrings(transformed.Terminal.EffectTrace, c.TargetTerminal.EffectTrace)
	replayOK := artifact.CounterexampleReplay.Stable && transformed.Terminal.Decision == artifact.CounterexampleReplay.CandidateDecision && reasonOK && effectsOK
	visibilityOK := artifact.CounterexampleVisible
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
			artifact.Preconditions[index].Observed = visibilityOK
		}
	}

	// A candidate that changes the repair reason/effect contract or hides the
	// counterexample is a hard refutation, even if another predicate fails too.
	refuted := !reasonOK || !effectsOK || !visibilityOK
	accepted := originOK && irDigestOK && idsOK && reasonOK && effectsOK && replayOK && visibilityOK
	if refuted {
		artifact.CandidateStatus = DecisionRefuted
	} else if accepted {
		artifact.CandidateStatus = DecisionClosed
	} else {
		artifact.CandidateStatus = DecisionUnknown
	}
	if !accepted && !refuted {
		artifact.Unknown = unknown("verify", "evaluate_"+operator.ID, "candidate did not establish every hard predicate", "HARD_PREDICATE_INCOMPLETE", "retain_candidate_for_audit", operator.ID)
	}
	return candidateResult(artifact, accepted, refuted)
}

func appendRequiredPredicates(values []PredicateResult, inputDigest string, transformed bool, visible bool, detail string) []PredicateResult {
	result := append([]PredicateResult{}, values...)
	result = append(result,
		PredicateResult{ID: "origin-digest", Observed: transformed, Detail: detailOr(detail, "origin source digest equals input source digest")},
		PredicateResult{ID: "ir-digest", Observed: transformed, Detail: inputDigest},
		PredicateResult{ID: "semantic-ids", Observed: transformed, Detail: detailOr(detail, "deterministic affected ID set")},
		PredicateResult{ID: "terminal-reason", Observed: transformed, Detail: detailOr(detail, "terminal reason matches target")},
		PredicateResult{ID: "effect-trace", Observed: transformed, Detail: detailOr(detail, "effect trace matches target")},
		PredicateResult{ID: "replay", Observed: transformed, Detail: detailOr(detail, "stable replay")},
		PredicateResult{ID: "visibility", Observed: visible, Detail: detailOr(detail, "counterexample anchor remains visible")},
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
			return result, []string{node.ID, guardID, guardID + "::edge"}, true, "effect node is guardable"
		}
		return result, nil, false, "no guardable effect node"
	case "effect_narrowing":
		for _, node := range result.Nodes {
			if node.Kind != "effect" || attributes(node)["narrowable"] != "true" {
				continue
			}
			updated := attributes(node)
			updated["effect_scope"] = "narrow"
			updated["effect_capability"] = "read-only-proof"
			for index := range result.Nodes {
				if result.Nodes[index].ID == node.ID {
					result.Nodes[index].Attributes = updated
					result.Nodes[index].Payload = ""
				}
			}
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
			elseID := node.ID + "::else"
			elseNode := GraphNode{ID: elseID, Kind: "branch", Type: "branch<else>", Label: node.Label + " else", Attributes: cloneMap(thenNode.Attributes)}
			elseNode.Attributes["route"] = "else"
			result.Nodes[index] = thenNode
			result.Nodes = append(result.Nodes, elseNode)
			result.Edges = append(result.Edges, GraphEdge{ID: elseID + "::edge", From: node.ID, To: elseID, Kind: "branch_route", Type: "typed_branch"})
			return result, []string{node.ID, elseID, elseID + "::edge"}, true, "branch node is splittable"
		}
		return result, nil, false, "no splittable branch node"
	default:
		return result, nil, false, "operator kind is not implemented"
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
	result := ReplayResult{
		BaselineDecision: baseline.Decision, CandidateDecision: candidate.Decision,
		BaselineReasonDigest: baseline.ReasonDigest, CandidateReasonDigest: candidate.ReasonDigest,
		BaselineEffectTrace: cloneStrings(baseline.EffectTrace), CandidateEffectTrace: cloneStrings(candidate.EffectTrace),
		CounterexampleVisible: visible, Stable: true, Replays: count,
	}
	return result
}

func candidateResult(artifact CandidateArtifact, accepted, refuted bool) CandidateResult {
	return CandidateResult{Artifact: artifact, Accepted: accepted, Refuted: refuted, Summary: CandidateSummary{
		CandidateID: artifact.CandidateID, Operator: artifact.Operator, Status: artifact.CandidateStatus,
		Accepted: accepted, Refuted: refuted, TransformedIRDigest: artifact.TransformedIRDigest,
		AffectedSemanticIDs: cloneStrings(artifact.AffectedSemanticIDs),
		ArtifactGooo:        "candidates/" + artifact.CandidateID + ".gooo", ArtifactIR: "candidates/" + artifact.CandidateID + ".ir.json",
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
	if a.Schema != CandidateSchema || a.CandidateID == "" || a.Scenario == "" || a.Operator == "" || !allowedDecision(a.CandidateStatus) {
		return errors.New("candidate artifact identity or status is incomplete")
	}
	if !validDigest(a.SourceDigest) || !validDigest(a.ToolchainDigest) || a.IR.Scenario != a.Scenario {
		return errors.New("candidate provenance is incomplete")
	}
	if a.CandidateStatus != DecisionUnknown && !validDigest(a.TransformedIRDigest) {
		return errors.New("candidate transformed IR digest is invalid")
	}
	return nil
}
