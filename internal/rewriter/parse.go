package rewriter

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func LoadMeta(path string) (MetaContract, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return MetaContract{}, nil, err
	}
	meta, err := parseMeta(string(raw))
	if err != nil {
		return MetaContract{}, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := meta.Validate(); err != nil {
		return MetaContract{}, nil, err
	}
	meta.ContractDigest = Digest(raw)
	return meta, raw, nil
}

func parseMeta(input string) (MetaContract, error) {
	var meta MetaContract
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	for lineNumber, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens, err := tokenize(line)
		if err != nil {
			return MetaContract{}, fmt.Errorf("line %d: %w", lineNumber+1, err)
		}
		if len(tokens) == 0 {
			continue
		}
		values, err := keyValues(tokens[1:])
		if err != nil && tokens[0] != "gooo" && tokens[0] != "authority" && tokens[0] != "precedence" && tokens[0] != "unknown_fields" && tokens[0] != "candidate_order" {
			return MetaContract{}, fmt.Errorf("line %d: %w", lineNumber+1, err)
		}
		switch tokens[0] {
		case "gooo":
			if len(tokens) != 3 || tokens[1] != "counterexample_guided_rewriter" || tokens[2] != "v1" {
				return MetaContract{}, fmt.Errorf("line %d: invalid header", lineNumber+1)
			}
			meta.Schema = MetaSchema
		case "authority":
			if len(tokens) != 2 {
				return MetaContract{}, fmt.Errorf("line %d: invalid authority", lineNumber+1)
			}
			meta.Authority = tokens[1]
		case "denominator":
			meta.Denominator = DenominatorDecl{ID: values["id"], Scenarios: mustInt(values["scenarios"]), Unit: values["unit"]}
		case "decision":
			meta.Statuses = splitCSV(values["statuses"])
		case "precedence":
			if len(tokens) != 2 {
				return MetaContract{}, fmt.Errorf("line %d: invalid precedence", lineNumber+1)
			}
			meta.Precedence = strings.Split(tokens[1], ">")
		case "unknown_fields":
			if len(tokens) != 2 {
				return MetaContract{}, fmt.Errorf("line %d: invalid unknown_fields", lineNumber+1)
			}
			meta.UnknownFields = strings.Split(tokens[1], ",")
		case "source_policy":
			meta.SourcePolicy = values
		case "toolchain":
			meta.Toolchain = ToolchainDecl{Go: values["go"], Digest: values["digest"]}
		case "search":
			meta.Search = SearchDecl{Bound: mustInt(values["bound"]), Order: splitCSV(values["order"])}
		case "replay_replays":
			meta.ReplayReplays = mustInt(values["count"])
		case "cross_project_gate":
			meta.CrossProjectGate = mustInt(values["required"])
		case "predicate":
			meta.Predicates = append(meta.Predicates, PredicateDecl{Ordinal: mustInt(values["ordinal"]), ID: values["id"], Kind: values["kind"], Preserve: mustBool(values["preserve"])})
		case "operator":
			meta.Operators = append(meta.Operators, OperatorDecl{Ordinal: mustInt(values["ordinal"]), ID: values["id"], Kind: values["kind"], Target: values["target"], Cost: mustInt(values["cost"])})
		case "oracle":
			meta.Oracles = append(meta.Oracles, OraclePin{Name: values["name"], Repository: values["repository"], Release: values["release"], Digest: values["digest"], Required: mustBool(values["required"])})
		case "generation_plan":
			meta.GenerationPlan = GenerationPlan{Order: splitCSV(values["order"]), Outputs: splitCSV(values["outputs"])}
		case "candidate_space":
			meta.CandidateSpace = CandidateSpaceDecl{ID: values["id"], Digest: values["digest"], Bound: mustInt(values["bound"])}
		case "rewrite_rule":
			meta.Rule = RuleDecl{ID: values["id"], Digest: values["digest"], Count: mustInt(values["count"])}
		case "evaluator":
			meta.Evaluator = EvaluatorDecl{ID: values["id"], Digest: values["digest"], Kind: values["kind"]}
		case "acceptance":
			meta.Acceptance = AcceptanceDecl{Relation: values["relation"], Requires: splitCSV(values["requires"])}
		case "meta_activity":
			meta.MetaActivities = append(meta.MetaActivities, MetaActivity{Ordinal: mustInt(values["ordinal"]), ID: values["id"], Kind: values["kind"], Expected: values["expected"]})
		case "proof_counts":
			meta.ProofCounts = DecisionCounts{Closed: mustInt(values["closed"]), Unknown: mustInt(values["unknown"]), Refuted: mustInt(values["refuted"])}
		case "indicator_counts":
			meta.IndicatorCounts = DecisionCounts{Closed: mustInt(values["closed"]), Unknown: mustInt(values["unknown"]), Refuted: mustInt(values["refuted"])}
		case "scenario":
			meta.Scenarios = append(meta.Scenarios, ScenarioDecl{Ordinal: mustInt(values["ordinal"]), ID: values["id"], Fixture: values["fixture"], Expected: values["expected"], ProofChoice: values["proof_choice"], IndicatorClass: values["indicator_class"]})
		default:
			return MetaContract{}, fmt.Errorf("line %d: unknown declaration %q", lineNumber+1, tokens[0])
		}
	}
	return meta, nil
}

func LoadCounterexample(path string) (Counterexample, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Counterexample{}, nil, err
	}
	var record struct {
		Schema             string `json:"schema"`
		ID                 string `json:"id"`
		Scenario           string `json:"scenario"`
		SourceDigest       string `json:"source_digest"`
		ToolchainDigest    string `json:"toolchain_digest"`
		OriginSourceDigest string `json:"origin_source_digest"`
		OriginAnchor       string `json:"origin_anchor"`
		Origin             struct {
			SourceDigest string `json:"source_digest"`
			Anchor       string `json:"anchor"`
		} `json:"origin"`
		Baseline       TerminalTrace  `json:"baseline"`
		TargetTerminal TerminalTrace  `json:"target_terminal"`
		ReducedGraph   Graph          `json:"reduced_graph"`
		Replay         ReplayEvidence `json:"replay"`
		ContractDigest string          `json:"contract_digest"`
		CandidateSpaceID string        `json:"candidate_space_id"`
		CandidateSpaceDigest string    `json:"candidate_space_digest"`
		RuleID         string          `json:"rule_id"`
		RuleDigest     string          `json:"rule_digest"`
		EvaluatorID    string          `json:"evaluator_id"`
		EvaluatorDigest string         `json:"evaluator_digest"`
		CausalInput    CausalInput     `json:"causal_input"`
		Corpus         []CorpusCase    `json:"corpus"`
		ExternalUtility ExternalUtility `json:"external_utility"`
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		return Counterexample{}, nil, fmt.Errorf("parse counterexample: %w", err)
	}
	if record.Schema != CounterexampleSchema && record.Schema != CounterexampleAlias {
		return Counterexample{}, nil, fmt.Errorf("unsupported counterexample schema %q", record.Schema)
	}
	if record.ID == "" {
		record.ID = record.Scenario
	}
	if record.OriginSourceDigest == "" {
		record.OriginSourceDigest = record.Origin.SourceDigest
	}
	if record.OriginAnchor == "" {
		record.OriginAnchor = record.Origin.Anchor
	}
	record.Baseline = record.Baseline.Normalized()
	record.TargetTerminal = record.TargetTerminal.Normalized()
	if record.Baseline.ReasonDigest == "" && record.Baseline.Reason != "" {
		record.Baseline.ReasonDigest = DigestString(record.Baseline.Reason)
	}
	counterexample := Counterexample{
		Schema: record.Schema, ID: record.ID, Scenario: record.Scenario, SourceDigest: record.SourceDigest, ToolchainDigest: record.ToolchainDigest,
		OriginSourceDigest: record.OriginSourceDigest, OriginAnchor: record.OriginAnchor, Baseline: record.Baseline,
		TargetTerminal: record.TargetTerminal, ReducedGraph: record.ReducedGraph, Replay: record.Replay,
		ContractDigest: record.ContractDigest, CandidateSpaceID: record.CandidateSpaceID, CandidateSpaceDigest: record.CandidateSpaceDigest,
		RuleID: record.RuleID, RuleDigest: record.RuleDigest, EvaluatorID: record.EvaluatorID, EvaluatorDigest: record.EvaluatorDigest,
		CausalInput: record.CausalInput, Corpus: record.Corpus, ExternalUtility: record.ExternalUtility,
	}
	if err := counterexample.Validate(); err != nil {
		return Counterexample{}, nil, err
	}
	return counterexample, raw, nil
}

func (c Counterexample) Validate() error {
	if (c.Schema != CounterexampleSchema && c.Schema != CounterexampleAlias) || c.ID == "" || c.Scenario == "" || c.SourceDigest == "" || c.ToolchainDigest == "" {
		return fmt.Errorf("counterexample identity is incomplete")
	}
	if !validDigest(c.SourceDigest) || !validDigest(c.ToolchainDigest) {
		return fmt.Errorf("counterexample source or toolchain digest is invalid")
	}
	if (c.Baseline.Decision != DecisionRefuted && c.Baseline.Decision != DecisionUnknown) || c.Baseline.Reason == "" || !validDigest(c.Baseline.ReasonDigest) || c.Baseline.EffectTrace == nil {
		return fmt.Errorf("counterexample baseline must be an UNKNOWN or REFUTED terminal with reason and effect trace")
	}
	if err := c.ReducedGraph.Validate(); err != nil {
		return err
	}
	return nil
}

func (c Counterexample) HasOrigin() bool {
	return validDigest(c.OriginSourceDigest) && c.OriginSourceDigest == c.SourceDigest && c.OriginAnchor != ""
}

func (c Counterexample) HasTargetEvidence() bool {
	return c.TargetTerminal.Valid()
}

func (c Counterexample) IdentityFields() map[string]string {
	return map[string]string{
		"candidate-space-id": c.CandidateSpaceID, "candidate-space-digest": c.CandidateSpaceDigest,
		"rule-id": c.RuleID, "rule-digest": c.RuleDigest,
		"evaluator-id": c.EvaluatorID, "evaluator-digest": c.EvaluatorDigest,
	}
}

func (c Counterexample) HasCausalInput() bool {
	return c.CausalInput.Valid()
}

func (c Counterexample) HasCorpus() bool {
	if len(c.Corpus) == 0 {
		return false
	}
	hasNormal, hasRegression := false, false
	for _, item := range c.Corpus {
		if item.ID == "" || item.Input == "" || (item.Class != "normal" && item.Class != "regression" && item.Class != "counterexample") || !item.Before.Valid() || !item.After.Valid() {
			return false
		}
		hasNormal = hasNormal || item.Class == "normal"
		hasRegression = hasRegression || item.Class == "regression"
	}
	return hasNormal && hasRegression
}

func (c Counterexample) AnchorVisible(ir SemanticIR) bool {
	if c.OriginAnchor == "" {
		return false
	}
	for _, node := range ir.Nodes {
		if node.ID == c.OriginAnchor {
			return true
		}
	}
	for _, edge := range ir.Edges {
		if edge.ID == c.OriginAnchor {
			return true
		}
	}
	return false
}

func (g Graph) ToIR(c Counterexample) SemanticIR {
	return SemanticIR{
		Schema: IRSchema, Scenario: c.Scenario, OriginSourceDigest: c.OriginSourceDigest, ContractDigest: c.ContractDigest,
		ToolchainDigest: c.ToolchainDigest, CandidateSpaceID: c.CandidateSpaceID, CandidateSpaceDigest: c.CandidateSpaceDigest,
		RuleID: c.RuleID, RuleDigest: c.RuleDigest, EvaluatorID: c.EvaluatorID, EvaluatorDigest: c.EvaluatorDigest,
		Nodes: cloneNodes(g.Nodes), Edges: cloneEdges(g.Edges), Terminal: c.Baseline.Normalized(),
	}
}

func cloneNodes(values []GraphNode) []GraphNode {
	result := make([]GraphNode, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Attributes = cloneMap(value.Attributes)
	}
	return result
}

func cloneEdges(values []GraphEdge) []GraphEdge {
	result := make([]GraphEdge, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Attributes = cloneMap(value.Attributes)
	}
	return result
}

func cloneMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func attributes(node GraphNode) map[string]string {
	result := cloneMap(node.Attributes)
	if result == nil {
		result = map[string]string{}
	}
	if node.Payload != "" {
		for _, part := range strings.FieldsFunc(node.Payload, func(r rune) bool { return r == ';' || r == ',' || r == ' ' }) {
			key, value, ok := strings.Cut(part, "=")
			if ok && key != "" && value != "" {
				if _, exists := result[key]; !exists {
					result[key] = value
				}
			}
		}
	}
	return result
}

func tokenize(line string) ([]string, error) {
	var tokens []string
	for index := 0; index < len(line); {
		for index < len(line) && (line[index] == ' ' || line[index] == '\t') {
			index++
		}
		if index == len(line) {
			break
		}
		if line[index] == '"' {
			start := index
			index++
			for index < len(line) {
				if line[index] == '\\' {
					index += 2
					continue
				}
				if line[index] == '"' {
					index++
					break
				}
				index++
			}
			if index > len(line) || line[index-1] != '"' {
				return nil, fmt.Errorf("unterminated quoted value")
			}
			value, err := strconv.Unquote(line[start:index])
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, value)
			continue
		}
		start := index
		for index < len(line) && line[index] != ' ' && line[index] != '\t' {
			index++
		}
		tokens = append(tokens, line[start:index])
	}
	return tokens, nil
}

func keyValues(tokens []string) (map[string]string, error) {
	values := map[string]string{}
	for _, token := range tokens {
		key, value, ok := strings.Cut(token, "=")
		if !ok || key == "" || value == "" || values[key] != "" {
			return nil, fmt.Errorf("invalid key/value %q", token)
		}
		values[key] = value
	}
	return values, nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func mustInt(value string) int { result, _ := strconv.Atoi(value); return result }

func mustBool(value string) bool { result, _ := strconv.ParseBool(value); return result }
