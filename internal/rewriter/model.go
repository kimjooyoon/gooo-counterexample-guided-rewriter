package rewriter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	MetaSchema           = "gooo.counterexample_guided_rewriter/v1"
	CounterexampleSchema = "gooo/semantic_counterexample_reduction_report/v1"
	CounterexampleAlias  = "gooo/counterexample-guided-rewriter/counterexample/v1"
	GraphSchema          = "gooo/semantic_graph/v1"
	IRSchema             = "gooo/counterexample-guided-rewriter/typed-ir/v1"
	CandidateSchema      = "gooo/counterexample-guided-rewriter/candidate/v1"
	CaseReportSchema     = "gooo/counterexample-guided-rewriter/case-report/v1"
	ConformanceSchema    = "gooo/counterexample-guided-rewriter/conformance/v1"
	MetricsSchema        = "gooo/counterexample-guided-rewriter/metrics/v1"
	DecisionClosed       = "CLOSED"
	DecisionUnknown      = "UNKNOWN"
	DecisionRefuted      = "REFUTED"
)

var requiredUnknownFields = []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}
var requiredOperators = []string{"guard-insertion", "effect-narrowing", "reason-preserving-branch-split"}
var requiredPredicates = []string{"precondition", "origin-digest", "ir-digest", "semantic-ids", "terminal-reason", "effect-trace", "replay", "visibility"}

type MetaContract struct {
	Schema           string
	Authority        string
	Denominator      DenominatorDecl
	Statuses         []string
	Precedence       []string
	UnknownFields    []string
	SourcePolicy     map[string]string
	Toolchain        ToolchainDecl
	Search           SearchDecl
	ReplayReplays    int
	CrossProjectGate int
	Predicates       []PredicateDecl
	Operators        []OperatorDecl
	Oracles          []OraclePin
	GenerationPlan   GenerationPlan
	Scenarios        []ScenarioDecl
}

type DenominatorDecl struct {
	ID        string `json:"id"`
	Scenarios int    `json:"scenarios"`
	Unit      string `json:"unit"`
}

type ToolchainDecl struct {
	Go     string `json:"go"`
	Digest string `json:"digest"`
}

type SearchDecl struct {
	Bound int      `json:"bound"`
	Order []string `json:"order"`
}

type PredicateDecl struct {
	Ordinal  int    `json:"ordinal"`
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Preserve bool   `json:"preserve"`
}

type OperatorDecl struct {
	Ordinal int    `json:"ordinal"`
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Target  string `json:"target"`
	Cost    int    `json:"cost"`
}

type OraclePin struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	Release    string `json:"release"`
	Digest     string `json:"digest"`
	Required   bool   `json:"required"`
}

type GenerationPlan struct {
	Order   []string `json:"order"`
	Outputs []string `json:"outputs"`
}

type ScenarioDecl struct {
	Ordinal  int    `json:"ordinal"`
	ID       string `json:"id"`
	Fixture  string `json:"fixture"`
	Expected string `json:"expected"`
}

type TerminalTrace struct {
	Decision              string   `json:"decision"`
	Reason                string   `json:"reason"`
	ReasonDigest          string   `json:"reason_digest"`
	EffectTrace           []string `json:"effect_trace"`
	CounterexampleVisible bool     `json:"counterexample_visible"`
}

type Graph struct {
	Schema string      `json:"schema"`
	Nodes  []GraphNode `json:"nodes"`
	Edges  []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Type       string            `json:"type,omitempty"`
	Label      string            `json:"label"`
	Payload    string            `json:"payload,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type GraphEdge struct {
	ID         string            `json:"id"`
	From       string            `json:"from"`
	To         string            `json:"to"`
	Kind       string            `json:"kind"`
	Type       string            `json:"type,omitempty"`
	Payload    string            `json:"payload,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type SemanticIR struct {
	Schema             string        `json:"schema"`
	Scenario           string        `json:"scenario"`
	OriginSourceDigest string        `json:"origin_source_digest"`
	ToolchainDigest    string        `json:"toolchain_digest"`
	Nodes              []GraphNode   `json:"nodes"`
	Edges              []GraphEdge   `json:"edges"`
	Terminal           TerminalTrace `json:"terminal"`
}

type ReplayEvidence struct {
	BaselineReplays  int `json:"baseline_replays"`
	CandidateReplays int `json:"candidate_replays"`
}

type Counterexample struct {
	Schema             string         `json:"schema"`
	ID                 string         `json:"id"`
	Scenario           string         `json:"scenario"`
	SourceDigest       string         `json:"source_digest"`
	ToolchainDigest    string         `json:"toolchain_digest"`
	OriginSourceDigest string         `json:"origin_source_digest"`
	OriginAnchor       string         `json:"origin_anchor"`
	Baseline           TerminalTrace  `json:"baseline"`
	TargetTerminal     TerminalTrace  `json:"target_terminal"`
	ReducedGraph       Graph          `json:"reduced_graph"`
	Replay             ReplayEvidence `json:"replay"`
}

type PredicateResult struct {
	ID       string `json:"id"`
	Observed bool   `json:"observed"`
	Detail   string `json:"detail"`
}

type ReplayResult struct {
	BaselineDecision      string   `json:"baseline_decision"`
	CandidateDecision     string   `json:"candidate_decision"`
	BaselineReasonDigest  string   `json:"baseline_reason_digest"`
	CandidateReasonDigest string   `json:"candidate_reason_digest"`
	BaselineEffectTrace   []string `json:"baseline_effect_trace"`
	CandidateEffectTrace  []string `json:"candidate_effect_trace"`
	CounterexampleVisible bool     `json:"counterexample_visible"`
	Stable                bool     `json:"stable"`
	Replays               int      `json:"replays"`
}

type UnknownRecord struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type CandidateArtifact struct {
	Schema                string            `json:"schema"`
	CandidateID           string            `json:"candidate_id"`
	Scenario              string            `json:"scenario"`
	Operator              string            `json:"operator"`
	CandidateStatus       string            `json:"candidate_status"`
	SourceDigest          string            `json:"source_digest"`
	OriginSourceDigest    string            `json:"origin_source_digest"`
	ToolchainDigest       string            `json:"toolchain_digest"`
	InputIRDigest         string            `json:"input_ir_digest"`
	TransformedIRDigest   string            `json:"transformed_ir_digest"`
	Preconditions         []PredicateResult `json:"preconditions"`
	AffectedSemanticIDs   []string          `json:"affected_semantic_ids"`
	ExpectedTerminal      TerminalTrace     `json:"expected_terminal"`
	CounterexampleReplay  ReplayResult      `json:"counterexample_replay"`
	CounterexampleVisible bool              `json:"counterexample_visible"`
	IR                    SemanticIR        `json:"ir"`
	Unknown               *UnknownRecord    `json:"unknown,omitempty"`
}

type CandidateSummary struct {
	CandidateID         string   `json:"candidate_id"`
	Operator            string   `json:"operator"`
	Status              string   `json:"status"`
	Accepted            bool     `json:"accepted"`
	Refuted             bool     `json:"refuted"`
	TransformedIRDigest string   `json:"transformed_ir_digest"`
	AffectedSemanticIDs []string `json:"affected_semantic_ids"`
	ArtifactGooo        string   `json:"artifact_gooo"`
	ArtifactIR          string   `json:"artifact_ir"`
}

type CaseReport struct {
	Schema               string             `json:"schema"`
	Scenario             string             `json:"scenario"`
	Decision             string             `json:"decision"`
	ExpectedDecision     string             `json:"expected_decision"`
	SourceDigest         string             `json:"source_digest"`
	OriginSourceDigest   string             `json:"origin_source_digest"`
	ToolchainDigest      string             `json:"toolchain_digest"`
	InputIRDigest        string             `json:"input_ir_digest"`
	Baseline             TerminalTrace      `json:"baseline"`
	TargetTerminal       TerminalTrace      `json:"target_terminal"`
	Candidates           []CandidateSummary `json:"candidates"`
	AcceptedCandidateIDs []string           `json:"accepted_candidate_ids"`
	Unknown              *UnknownRecord     `json:"unknown,omitempty"`
	RepositoryWrites     int                `json:"repository_writes"`
}

type ConformanceCase struct {
	Scenario string `json:"scenario"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
	Pass     bool   `json:"pass"`
}

type ConformanceReport struct {
	Schema           string            `json:"schema"`
	Authority        string            `json:"authority"`
	MetaDigest       string            `json:"meta_digest"`
	Decision         string            `json:"decision"`
	Scenarios        int               `json:"scenarios"`
	Closed           int               `json:"closed"`
	Unknown          int               `json:"unknown"`
	Refuted          int               `json:"refuted"`
	RepositoryWrites int               `json:"repository_writes"`
	Cases            []ConformanceCase `json:"cases"`
}

type Metrics struct {
	Schema         string `json:"schema"`
	GoFiles        int    `json:"go_files"`
	GoooFiles      int    `json:"gooo_files"`
	PhysicalLines  int    `json:"physical_lines"`
	DescendantDirs int    `json:"descendant_dirs"`
	RegularFiles   int    `json:"regular_files"`
	GeneratedFiles int    `json:"generated_files"`
	GeneratedBytes int64  `json:"generated_bytes"`
	WallMS         int64  `json:"wall_ms"`
	PeakRSSKIB     int64  `json:"peak_rss_kib"`
	TestsTotal     int    `json:"tests_total"`
	TestsSelected  int    `json:"tests_selected"`
	TestsExecuted  int    `json:"tests_executed"`
	TestsReused    int    `json:"tests_reused"`
	TestsFailed    int    `json:"tests_failed"`
	TestsUnknown   int    `json:"tests_unknown"`
}

func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestString(value string) string { return Digest([]byte(value)) }

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func idsOfPredicates(values []PredicateDecl) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}

func idsOfOperators(values []OperatorDecl) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}

func allowedDecision(value string) bool {
	return value == DecisionClosed || value == DecisionUnknown || value == DecisionRefuted
}

func reduceDecision(values ...string) string {
	for _, value := range values {
		if value == DecisionRefuted {
			return DecisionRefuted
		}
	}
	for _, value := range values {
		if value == DecisionUnknown {
			return DecisionUnknown
		}
	}
	return DecisionClosed
}

func (m MetaContract) Validate() error {
	if m.Schema != MetaSchema || m.Authority != "metacode" {
		return errors.New(".gooo must declare the rewriter schema and metacode authority")
	}
	if m.Denominator.ID == "" || m.Denominator.Unit != "counterexample" || m.Denominator.Scenarios != len(m.Scenarios) || m.Denominator.Scenarios < 7 {
		return errors.New("denominator must match at least seven counterexample scenarios")
	}
	if !sameStrings(m.Statuses, []string{DecisionClosed, DecisionUnknown, DecisionRefuted}) || !sameStrings(m.Precedence, []string{DecisionRefuted, DecisionUnknown, DecisionClosed}) {
		return errors.New("decision statuses or precedence are not declared exactly")
	}
	if !sameStrings(m.UnknownFields, requiredUnknownFields) {
		return errors.New("UNKNOWN must declare the required six fields in order")
	}
	if m.SourcePolicy["input"] != "immutable_digest" || m.SourcePolicy["output"] != "caller_owned_temp" || m.SourcePolicy["repository_writes"] != "zero" {
		return errors.New("input, output, or repository write policy is incomplete")
	}
	if m.Toolchain.Go != "1.27" || !validDigest(m.Toolchain.Digest) {
		return errors.New("Go 1.27 toolchain digest is incomplete")
	}
	if m.Search.Bound != 1 || !sameStrings(m.Search.Order, requiredOperators) || m.ReplayReplays < 2 || m.CrossProjectGate != 0 {
		return errors.New("bounded search, replay, or cross-project gate is not declared exactly")
	}
	if !sameStrings(idsOfPredicates(m.Predicates), requiredPredicates) {
		return errors.New("hard predicates are incomplete or out of order")
	}
	for _, predicate := range m.Predicates {
		if predicate.Ordinal < 1 || predicate.Kind == "" || !predicate.Preserve {
			return fmt.Errorf("predicate %q is not preserved", predicate.ID)
		}
	}
	if !sameStrings(idsOfOperators(m.Operators), requiredOperators) || len(m.Operators) != 3 {
		return errors.New("the three required rewrite operators are not declared in order")
	}
	for _, operator := range m.Operators {
		if operator.Ordinal < 1 || operator.Kind == "" || operator.Target == "" || operator.Cost < 1 {
			return fmt.Errorf("operator %q is incomplete", operator.ID)
		}
	}
	for _, oracle := range m.Oracles {
		if oracle.Name == "" || oracle.Repository == "" || oracle.Release != "v0.1.1" || !validDigest(oracle.Digest) || oracle.Required {
			return fmt.Errorf("optional oracle %q is not a valid digest-pinned v0.1.1 pin", oracle.Name)
		}
	}
	if m.GenerationPlan.Order == nil || len(m.GenerationPlan.Outputs) < 4 {
		return errors.New("generation plan must declare candidate and report outputs")
	}
	seen := map[string]bool{}
	for _, scenario := range m.Scenarios {
		if scenario.ID == "" || scenario.Fixture == "" || seen[scenario.ID] || !allowedDecision(scenario.Expected) {
			return fmt.Errorf("scenario %q is incomplete or duplicated", scenario.ID)
		}
		seen[scenario.ID] = true
	}
	return nil
}

func (m MetaContract) Scenario(id string) (ScenarioDecl, error) {
	for _, scenario := range m.Scenarios {
		if scenario.ID == id {
			return scenario, nil
		}
	}
	return ScenarioDecl{}, fmt.Errorf("scenario %q is not declared in .gooo", id)
}

func (t TerminalTrace) Normalized() TerminalTrace {
	result := t
	if result.ReasonDigest == "" && result.Reason != "" {
		result.ReasonDigest = DigestString(result.Reason)
	}
	result.EffectTrace = append([]string(nil), t.EffectTrace...)
	return result
}

func (t TerminalTrace) Valid() bool {
	return allowedDecision(t.Decision) && t.Reason != "" && validDigest(t.ReasonDigest) && t.ReasonDigest == DigestString(t.Reason) && t.EffectTrace != nil
}

func (g Graph) Validate() error {
	if g.Schema != GraphSchema || len(g.Nodes) == 0 {
		return errors.New("typed semantic graph is incomplete")
	}
	seen := map[string]bool{}
	for _, node := range g.Nodes {
		if node.ID == "" || node.Kind == "" || seen[node.ID] {
			return fmt.Errorf("invalid or duplicate node %q", node.ID)
		}
		seen[node.ID] = true
	}
	for _, edge := range g.Edges {
		if edge.ID == "" || edge.From == "" || edge.To == "" || edge.Kind == "" || seen[edge.ID] {
			return fmt.Errorf("invalid or duplicate edge %q", edge.ID)
		}
		seen[edge.ID] = true
	}
	return nil
}

func (ir SemanticIR) CanonicalDigest() string {
	data, _ := json.Marshal(ir)
	return Digest(data)
}

func (ir SemanticIR) Validate() error {
	if ir.Schema != IRSchemasafe() || ir.Scenario == "" || !validDigest(ir.OriginSourceDigest) || !validDigest(ir.ToolchainDigest) || !ir.Terminal.Valid() {
		return errors.New("typed IR is incomplete")
	}
	graph := Graph{Schema: GraphSchema, Nodes: ir.Nodes, Edges: ir.Edges}
	return graph.Validate()
}

func IRSchemasafe() string { return IRSchemaconstant() }

func IRSchemaconstant() string { return IRSchema }

func cloneStrings(values []string) []string { return append([]string(nil), values...) }

func sortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	unique := result[:0]
	for _, value := range result {
		if value != "" && (len(unique) == 0 || unique[len(unique)-1] != value) {
			unique = append(unique, value)
		}
	}
	return unique
}
