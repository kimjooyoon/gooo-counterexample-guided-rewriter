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
	SemanticMetricsSchema = "gooo/counterexample-guided-rewriter/semantic-metrics-dossier/v1"
	DossierSchema        = "gooo/counterexample-guided-rewriter/candidate-dossier/v1"
	PatchSchema          = "gooo/counterexample-guided-rewriter/patch/v1"
	EvaluationSchema     = "gooo/counterexample-guided-rewriter/evaluation/v1"
	CandidateAccept      = "ACCEPT"
	DecisionClosed       = "CLOSED"
	DecisionUnknown      = "UNKNOWN"
	DecisionRefuted      = "REFUTED"
)

var requiredUnknownFields = []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}
var requiredOperators = []string{"guard-insertion", "effect-narrowing", "reason-preserving-branch-split"}
var requiredPredicates = []string{"precondition", "origin-digest", "ir-digest", "semantic-ids", "terminal-reason", "effect-trace", "replay", "visibility", "corpus-preserved", "identity"}
var requiredProofChoices = []string{"FOUNDATION", "COHERENCE", "REGRESSION"}
var requiredIndicatorClasses = []string{"DRIVER", "OUTCOME", "GUARDRAIL"}

type MetaContract struct {
	Schema           string
	Authority        string
	ContractDigest   string
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
	CandidateSpace   CandidateSpaceDecl
	Rule             RuleDecl
	Evaluator        EvaluatorDecl
	Acceptance       AcceptanceDecl
	MetaActivities   []MetaActivity
	ProofCounts      DecisionCounts
	IndicatorCounts  DecisionCounts
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

type CandidateSpaceDecl struct {
	ID    string `json:"id"`
	Digest string `json:"digest"`
	Bound int    `json:"bound"`
}

type RuleDecl struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
	Count  int    `json:"count"`
}

type EvaluatorDecl struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
	Kind   string `json:"kind"`
}

type AcceptanceDecl struct {
	Relation string `json:"relation"`
	Requires []string `json:"requires"`
}

type MetaActivity struct {
	Ordinal  int    `json:"ordinal"`
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Expected string `json:"expected"`
}

type DecisionCounts struct {
	Closed  int `json:"CLOSED"`
	Unknown int `json:"UNKNOWN"`
	Refuted int `json:"REFUTED"`
}

type ScenarioDecl struct {
	Ordinal        int    `json:"ordinal"`
	ID             string `json:"id"`
	Fixture        string `json:"fixture"`
	Expected       string `json:"expected"`
	ProofChoice    string `json:"proof_choice"`
	IndicatorClass string `json:"indicator_class"`
}

type CausalInput struct {
	ID               string        `json:"id"`
	Kind             string        `json:"kind"`
	ObservedTerminal TerminalTrace `json:"observed_terminal"`
	TargetTerminal   TerminalTrace `json:"target_terminal"`
}

func (input CausalInput) Valid() bool {
	return input.ID != "" && input.Kind == "counterexample" && input.ObservedTerminal.Valid() && input.TargetTerminal.Valid()
}

type CorpusCase struct {
	ID     string        `json:"id"`
	Class  string        `json:"class"`
	Input  string        `json:"input"`
	Before TerminalTrace `json:"before"`
	After  TerminalTrace `json:"after"`
}

type ExternalUtility struct {
	Name      string `json:"name"`
	Release   string `json:"release"`
	Digest    string `json:"digest"`
	Available bool   `json:"available"`
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
	Schema               string        `json:"schema"`
	Scenario             string        `json:"scenario"`
	ProofChoice          string        `json:"proof_choice"`
	IndicatorClass       string        `json:"indicator_class"`
	OriginSourceDigest   string        `json:"origin_source_digest"`
	ContractDigest       string        `json:"contract_digest"`
	ToolchainDigest      string        `json:"toolchain_digest"`
	CandidateSpaceID     string        `json:"candidate_space_id"`
	CandidateSpaceDigest string        `json:"candidate_space_digest"`
	RuleID               string        `json:"rule_id"`
	RuleDigest           string        `json:"rule_digest"`
	EvaluatorID          string        `json:"evaluator_id"`
	EvaluatorDigest      string        `json:"evaluator_digest"`
	Nodes                []GraphNode   `json:"nodes"`
	Edges                []GraphEdge   `json:"edges"`
	Terminal             TerminalTrace `json:"terminal"`
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
	ContractDigest     string         `json:"contract_digest"`
	CandidateSpaceID   string         `json:"candidate_space_id"`
	CandidateSpaceDigest string       `json:"candidate_space_digest"`
	RuleID             string         `json:"rule_id"`
	RuleDigest         string         `json:"rule_digest"`
	EvaluatorID        string         `json:"evaluator_id"`
	EvaluatorDigest    string         `json:"evaluator_digest"`
	Baseline           TerminalTrace  `json:"baseline"`
	TargetTerminal     TerminalTrace  `json:"target_terminal"`
	CausalInput        CausalInput    `json:"causal_input"`
	Corpus             []CorpusCase   `json:"corpus"`
	ExternalUtility    ExternalUtility `json:"external_utility"`
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

type CorpusEvidence struct {
	InputID  string        `json:"input_id"`
	Class    string        `json:"class"`
	Before   TerminalTrace `json:"before"`
	After    TerminalTrace `json:"after"`
	ExpectedBefore TerminalTrace `json:"expected_before"`
	ExpectedAfter  TerminalTrace `json:"expected_after"`
	Unchanged bool          `json:"unchanged"`
	Preserved bool          `json:"preserved"`
}

type EvaluationIdentity struct {
	Fixture         string `json:"fixture"`
	SourceDigest    string `json:"source_digest"`
	ContractDigest  string `json:"contract_digest"`
	ToolchainDigest string `json:"toolchain_digest"`
	EvaluatorID     string `json:"evaluator_id"`
	EvaluatorDigest string `json:"evaluator_digest"`
}

type EvaluationResult struct {
	Schema                string                `json:"schema"`
	ProofChoice           string                `json:"proof_choice"`
	IndicatorClass        string                `json:"indicator_class"`
	Identity              EvaluationIdentity    `json:"identity"`
	IdentityMatch         bool                  `json:"identity_match"`
	CausalInputID         string                `json:"causal_input_id"`
	CounterexampleRemoved bool                  `json:"counterexample_removed"`
	CorpusPreserved       bool                  `json:"corpus_preserved"`
	KnownContradiction    bool                  `json:"known_contradiction"`
	Contradiction         string                `json:"contradiction,omitempty"`
	Evidence              []CorpusEvidence      `json:"evidence"`
	PairedIndicator       []string              `json:"paired_indicator"`
	Improvement           *ImprovementEvidence  `json:"improvement"`
	Unknown               *UnknownRecord        `json:"unknown,omitempty"`
}

type ImprovementEvidence struct {
	Status  string         `json:"status"`
	Before  *int64         `json:"before"`
	After   *int64         `json:"after"`
	Delta   *int64         `json:"delta"`
	Reason  string         `json:"reason"`
	Unknown *UnknownRecord `json:"unknown,omitempty"`
}

type PatchOperation struct {
	Kind       string   `json:"kind"`
	TargetIDs  []string `json:"target_ids"`
	RuleID     string   `json:"rule_id"`
	Description string  `json:"description"`
}

type PatchArtifact struct {
	Schema            string          `json:"schema"`
	CandidateID       string          `json:"candidate_id"`
	Scenario          string          `json:"scenario"`
	SourceDigest      string          `json:"source_digest"`
	InputIRDigest     string          `json:"input_ir_digest"`
	TransformedIRDigest string        `json:"transformed_ir_digest"`
	CallerOwned       bool            `json:"caller_owned"`
	AutoApply         bool            `json:"auto_apply"`
	RepositoryWrites  int             `json:"repository_writes"`
	Operations        []PatchOperation `json:"operations"`
}

type CandidateDossier struct {
	Schema            string            `json:"schema"`
	CandidateID       string            `json:"candidate_id"`
	Scenario          string            `json:"scenario"`
	ProofChoice       string            `json:"proof_choice"`
	IndicatorClass    string            `json:"indicator_class"`
	Operator          string            `json:"operator"`
	Decision          string            `json:"decision"`
	CandidateStatus   string            `json:"candidate_status"`
	CandidateSpaceID  string            `json:"candidate_space_id"`
	CandidateSpaceDigest string         `json:"candidate_space_digest"`
	RuleID            string            `json:"rule_id"`
	RuleDigest        string            `json:"rule_digest"`
	EvaluatorID       string            `json:"evaluator_id"`
	EvaluatorDigest   string            `json:"evaluator_digest"`
	CausalInput       CausalInput       `json:"causal_input"`
	Patch             PatchArtifact     `json:"patch"`
	Evaluation        EvaluationResult  `json:"evaluation"`
	Unknown           *UnknownRecord    `json:"unknown,omitempty"`
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
	ProofChoice           string            `json:"proof_choice"`
	IndicatorClass        string            `json:"indicator_class"`
	Operator              string            `json:"operator"`
	CandidateStatus       string            `json:"candidate_status"`
	SourceDigest          string            `json:"source_digest"`
	OriginSourceDigest    string            `json:"origin_source_digest"`
	ContractDigest        string            `json:"contract_digest"`
	ToolchainDigest       string            `json:"toolchain_digest"`
	CandidateSpaceID      string            `json:"candidate_space_id"`
	CandidateSpaceDigest  string            `json:"candidate_space_digest"`
	RuleID                string            `json:"rule_id"`
	RuleDigest            string            `json:"rule_digest"`
	EvaluatorID           string            `json:"evaluator_id"`
	EvaluatorDigest       string            `json:"evaluator_digest"`
	InputIRDigest         string            `json:"input_ir_digest"`
	TransformedIRDigest   string            `json:"transformed_ir_digest"`
	Preconditions         []PredicateResult `json:"preconditions"`
	AffectedSemanticIDs   []string          `json:"affected_semantic_ids"`
	CausalInput          CausalInput        `json:"causal_input"`
	ExpectedTerminal      TerminalTrace     `json:"expected_terminal"`
	CounterexampleReplay  ReplayResult      `json:"counterexample_replay"`
	CounterexampleVisible bool              `json:"counterexample_visible"`
	CounterexampleResolved bool            `json:"counterexample_resolved"`
	CorpusPreserved       bool              `json:"corpus_preserved"`
	Evaluation            EvaluationResult  `json:"evaluation"`
	Patch                 PatchArtifact     `json:"patch"`
	IR                    SemanticIR        `json:"ir"`
	Unknown               *UnknownRecord    `json:"unknown,omitempty"`
}

type CandidateSummary struct {
	CandidateID         string   `json:"candidate_id"`
	Operator            string   `json:"operator"`
	ProofChoice         string   `json:"proof_choice"`
	IndicatorClass      string   `json:"indicator_class"`
	Status              string   `json:"status"`
	Decision            string   `json:"decision"`
	Accepted            bool     `json:"accepted"`
	Refuted             bool     `json:"refuted"`
	TransformedIRDigest string   `json:"transformed_ir_digest"`
	AffectedSemanticIDs []string `json:"affected_semantic_ids"`
	ArtifactGooo        string   `json:"artifact_gooo"`
	ArtifactIR          string   `json:"artifact_ir"`
	ArtifactPatch       string   `json:"artifact_patch"`
	ArtifactDossier     string   `json:"artifact_dossier"`
}

type CaseReport struct {
	Schema               string             `json:"schema"`
	Scenario             string             `json:"scenario"`
	ProofChoice          string             `json:"proof_choice"`
	IndicatorClass       string             `json:"indicator_class"`
	Decision             string             `json:"decision"`
	ExpectedDecision     string             `json:"expected_decision"`
	SourceDigest         string             `json:"source_digest"`
	OriginSourceDigest   string             `json:"origin_source_digest"`
	ToolchainDigest      string             `json:"toolchain_digest"`
	InputIRDigest        string             `json:"input_ir_digest"`
	Baseline             TerminalTrace      `json:"baseline"`
	TargetTerminal       TerminalTrace      `json:"target_terminal"`
	CausalInputID        string             `json:"causal_input_id"`
	CausalInput           CausalInput        `json:"causal_input"`
	EvaluatorID          string             `json:"evaluator_id"`
	EvaluatorDigest      string             `json:"evaluator_digest"`
	Candidates           []CandidateSummary `json:"candidates"`
	AcceptedCandidateIDs []string           `json:"accepted_candidate_ids"`
	Unknown              *UnknownRecord     `json:"unknown,omitempty"`
	RepositoryWrites     int                `json:"repository_writes"`
	LocalTestExecutions  int                `json:"local_test_executions"`
	CrossProjectRequiredGates int           `json:"cross_project_required_gates"`
	PairedIndicatorVector []string          `json:"paired_indicator_vector"`
}

type ConformanceCase struct {
	Scenario       string         `json:"scenario"`
	Expected       string         `json:"expected"`
	Observed       string         `json:"observed"`
	ProofChoice    string         `json:"proof_choice"`
	IndicatorClass string         `json:"indicator_class"`
	Unknown        *UnknownRecord `json:"unknown,omitempty"`
	Pass           bool           `json:"pass"`
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
	Cells            int               `json:"cells"`
	ProofCounts      DecisionCounts    `json:"proof_counts"`
	IndicatorCounts  DecisionCounts    `json:"indicator_counts"`
	PairedIndicatorVector []string      `json:"paired_indicator_vector"`
	ProofChoiceCounts    map[string]int `json:"proof_choice_counts"`
	IndicatorClassCounts map[string]int `json:"indicator_class_counts"`
	UnknownRecords       []UnknownCaseEvidence `json:"unknown_records"`
	ExternalUtilityUnknown int          `json:"external_utility_unknown"`
	RepositoryWrites int               `json:"repository_writes"`
	LocalTestExecutions int            `json:"local_test_executions"`
	CrossProjectRequiredGates int      `json:"cross_project_required_gates"`
	Cases            []ConformanceCase `json:"cases"`
}

type UnknownCaseEvidence struct {
	Scenario string         `json:"scenario"`
	Record   UnknownRecord `json:"record"`
}

type SemanticMetricCount struct {
	Name    string `json:"name"`
	Total   int    `json:"total"`
	Closed  int    `json:"closed"`
	Unknown int    `json:"unknown"`
	Refuted int    `json:"refuted"`
}

type SemanticMetricsDossier struct {
	Schema                   string                `json:"schema"`
	MetaDigest               string                `json:"meta_digest"`
	Denominator              int                   `json:"denominator"`
	ProofChoices             []SemanticMetricCount `json:"proof_choices"`
	IndicatorClasses         []SemanticMetricCount `json:"indicator_classes"`
	Cases                    []ConformanceCase     `json:"cases"`
	UnknownRecords           []UnknownCaseEvidence `json:"unknown_records"`
	PairedIndicatorVector    []string              `json:"paired_indicator_vector"`
	Improvement              *ImprovementEvidence `json:"improvement"`
	RepositoryWrites         int                   `json:"repository_writes"`
	LocalTestExecutions      int                   `json:"local_test_executions"`
	CrossProjectRequiredGates int                  `json:"cross_project_required_gates"`
}

type Metrics struct {
	Schema         string `json:"schema"`
	InventoryExcludes []string `json:"inventory_excludes"`
	GoFiles        int    `json:"go_files"`
	GoooFiles      int    `json:"gooo_files"`
	PhysicalLines  int    `json:"physical_lines"`
	DescendantDirs int    `json:"descendant_dirs"`
	RegularFiles   int    `json:"regular_files"`
	GeneratedFiles int    `json:"generated_files"`
	GeneratedBytes int64  `json:"generated_bytes"`
	CandidateGoooFiles int `json:"candidate_gooo_files"`
	CandidateIRFiles int `json:"candidate_ir_files"`
	CandidatePatchFiles int `json:"candidate_patch_files"`
	CandidateDossierFiles int `json:"candidate_dossier_files"`
	CaseReportFiles int `json:"case_report_files"`
	CausalInputFiles int `json:"causal_input_files"`
	SemanticMetricsDossierFiles int `json:"semantic_metrics_dossier_files"`
	WallMS         int64  `json:"wall_ms"`
	PeakRSSKIB     int64  `json:"peak_rss_kib"`
	BuildMS        *int64 `json:"build_ms"`
	TestMS         *int64 `json:"test_ms"`
	CacheHits      *int64 `json:"cache_hits"`
	CacheMisses    *int64 `json:"cache_misses"`
	MetricsStatus  string `json:"metrics_status"`
	MetricsUnknown *UnknownRecord `json:"metrics_unknown,omitempty"`
	ActionsRunID   int64 `json:"actions_run_id,omitempty"`
	TestsTotal     int    `json:"tests_total"`
	TestsSelected  int    `json:"tests_selected"`
	TestsExecuted  int    `json:"tests_executed"`
	TestsReused    int    `json:"tests_reused"`
	TestsFailed    int    `json:"tests_failed"`
	TestsUnknown   int    `json:"tests_unknown"`
	ProofClosed    int    `json:"proof_closed"`
	ProofUnknown   int    `json:"proof_unknown"`
	ProofRefuted   int    `json:"proof_refuted"`
	IndicatorClosed int   `json:"indicator_closed"`
	IndicatorUnknown int  `json:"indicator_unknown"`
	IndicatorRefuted int   `json:"indicator_refuted"`
	ExternalUtilityUnknown int `json:"external_utility_unknown"`
	LocalTestExecutions int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
	PairedIndicatorVector []string `json:"paired_indicator_vector"`
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

func allowedProofChoice(value string) bool {
	return value == "FOUNDATION" || value == "COHERENCE" || value == "REGRESSION"
}

func allowedIndicatorClass(value string) bool {
	return value == "DRIVER" || value == "OUTCOME" || value == "GUARDRAIL"
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
	if m.Denominator.ID == "" || m.Denominator.Unit != "counterexample" || m.Denominator.Scenarios != len(m.Scenarios) || m.Denominator.Scenarios != 12 {
		return errors.New("denominator must match exactly twelve counterexample cells")
	}
	if !sameStrings(m.Statuses, []string{DecisionClosed, DecisionUnknown, DecisionRefuted}) || !sameStrings(m.Precedence, []string{DecisionRefuted, DecisionUnknown, DecisionClosed}) {
		return errors.New("decision statuses or precedence are not declared exactly")
	}
	if !sameStrings(m.UnknownFields, requiredUnknownFields) {
		return errors.New("UNKNOWN must declare the required six fields in order")
	}
	if m.SourcePolicy["input"] != "immutable_digest" || m.SourcePolicy["output"] != "caller_owned_temp" || m.SourcePolicy["repository_writes"] != "zero" || m.SourcePolicy["local_test_executions"] != "zero" || m.SourcePolicy["auto_apply"] != "forbidden" || m.SourcePolicy["git_integration"] != "forbidden" {
		return errors.New("input, output, or repository write policy is incomplete")
	}
	if m.Toolchain.Go != "1.27" || !validDigest(m.Toolchain.Digest) {
		return errors.New("Go 1.27 toolchain digest is incomplete")
	}
	if m.Search.Bound != 1 || !sameStrings(m.Search.Order, requiredOperators) || m.ReplayReplays < 2 || m.CrossProjectGate != 0 {
		return errors.New("bounded search, replay, or cross-project gate is not declared exactly")
	}
	if m.CandidateSpace.ID == "" || !validDigest(m.CandidateSpace.Digest) || m.CandidateSpace.Bound != 1 || m.Rule.ID == "" || !validDigest(m.Rule.Digest) || m.Rule.Count != 3 || m.Evaluator.ID == "" || !validDigest(m.Evaluator.Digest) || m.Evaluator.Kind != "independent-before-after" {
		return errors.New("candidate space, finite rule set, or independent evaluator identity is incomplete")
	}
	if m.Acceptance.Relation != "counterexample-removed-and-corpus-unchanged" || !sameStrings(m.Acceptance.Requires, []string{"counterexample-removed", "corpus-preserved", "identity-match", "replay-stable"}) {
		return errors.New("acceptance relation is not declared exactly")
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
	if m.GenerationPlan.Order == nil || len(m.GenerationPlan.Outputs) < 6 {
		return errors.New("generation plan must declare candidate and report outputs")
	}
	if len(m.MetaActivities) != 12 || m.ProofCounts.Closed != 4 || m.ProofCounts.Unknown != 4 || m.ProofCounts.Refuted != 4 || m.IndicatorCounts.Closed != 4 || m.IndicatorCounts.Unknown != 4 || m.IndicatorCounts.Refuted != 4 {
		return errors.New("the .gooo contract must declare twelve activities and 4/4/4 proof and indicator counts")
	}
	activityIDs := map[string]bool{}
	activityCounts := DecisionCounts{}
	for _, activity := range m.MetaActivities {
		if activity.Ordinal < 1 || activity.ID == "" || activity.Kind == "" || !allowedDecision(activity.Expected) || activityIDs[activity.ID] {
			return fmt.Errorf("meta activity %q is incomplete or duplicated", activity.ID)
		}
		activityIDs[activity.ID] = true
		switch activity.Expected {
		case DecisionClosed:
			activityCounts.Closed++
		case DecisionUnknown:
			activityCounts.Unknown++
		case DecisionRefuted:
			activityCounts.Refuted++
		}
	}
	if activityCounts != m.ProofCounts || activityCounts != m.IndicatorCounts {
		return errors.New("proof and indicator activity counts must both be 4/4/4")
	}
	seen := map[string]bool{}
	proofCounts := map[string]int{}
	indicatorCounts := map[string]int{}
	for _, scenario := range m.Scenarios {
		if scenario.ID == "" || scenario.Fixture == "" || seen[scenario.ID] || !allowedDecision(scenario.Expected) || !allowedProofChoice(scenario.ProofChoice) || !allowedIndicatorClass(scenario.IndicatorClass) {
			return fmt.Errorf("scenario %q is incomplete or duplicated", scenario.ID)
		}
		seen[scenario.ID] = true
		proofCounts[scenario.ProofChoice]++
		indicatorCounts[scenario.IndicatorClass]++
	}
	for _, choice := range requiredProofChoices {
		if proofCounts[choice] != 4 {
			return fmt.Errorf("proof choice %s must have exactly four cases", choice)
		}
	}
	for _, class := range requiredIndicatorClasses {
		if indicatorCounts[class] != 4 {
			return fmt.Errorf("indicator class %s must have exactly four cases", class)
		}
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
	if ir.Schema != IRSchemasafe() || ir.Scenario == "" || !allowedProofChoice(ir.ProofChoice) || !allowedIndicatorClass(ir.IndicatorClass) || !validDigest(ir.OriginSourceDigest) || !validDigest(ir.ContractDigest) || !validDigest(ir.ToolchainDigest) || ir.CandidateSpaceID == "" || !validDigest(ir.CandidateSpaceDigest) || ir.RuleID == "" || !validDigest(ir.RuleDigest) || ir.EvaluatorID == "" || !validDigest(ir.EvaluatorDigest) || !ir.Terminal.Valid() {
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
