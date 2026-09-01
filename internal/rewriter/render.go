package rewriter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func WriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func RenderCandidateGooo(a CandidateArtifact) string {
	var lines []string
	lines = append(lines,
		"gooo candidate_rewrite v1",
		"",
		"authority metacode",
		"candidate id="+a.CandidateID+" scenario="+a.Scenario+" operator="+a.Operator+" status="+a.CandidateStatus,
		"source_digest="+a.SourceDigest+" origin_source_digest="+a.OriginSourceDigest+" toolchain_digest="+a.ToolchainDigest,
		"input_ir_digest="+a.InputIRDigest+" transformed_ir_digest="+a.TransformedIRDigest,
		"affected_semantic_ids="+strings.Join(a.AffectedSemanticIDs, ","),
	)
	for _, predicate := range a.Preconditions {
		lines = append(lines, "predicate id="+predicate.ID+" observed="+strconv.FormatBool(predicate.Observed)+" detail="+strconv.Quote(predicate.Detail))
	}
	terminal := a.ExpectedTerminal.Normalized()
	lines = append(lines,
		"terminal decision="+terminal.Decision+" reason="+strconv.Quote(terminal.Reason)+" reason_digest="+terminal.ReasonDigest+" effect_trace="+strings.Join(terminal.EffectTrace, ","),
		"replay baseline_decision="+a.CounterexampleReplay.BaselineDecision+" candidate_decision="+a.CounterexampleReplay.CandidateDecision+" stable="+strconv.FormatBool(a.CounterexampleReplay.Stable)+" replays="+strconv.Itoa(a.CounterexampleReplay.Replays)+" counterexample_visible="+strconv.FormatBool(a.CounterexampleVisible),
		"ir_digest="+a.TransformedIRDigest,
	)
	if a.Unknown != nil {
		lines = append(lines, "unknown stage="+a.Unknown.Stage+" step="+a.Unknown.Step+" reason="+strconv.Quote(a.Unknown.Reason)+" unknown_class="+a.Unknown.UnknownClass+" next_operation="+a.Unknown.NextOperation+" blocked_by="+strings.Join(a.Unknown.BlockedBy, ","))
	}
	return strings.Join(lines, "\n") + "\n"
}

func WriteCaseArtifacts(outputDir string, report CaseReport, results []CandidateResult) error {
	caseDir := filepath.Join(outputDir, "cases", report.Scenario)
	if err := os.MkdirAll(filepath.Join(caseDir, "candidates"), 0o755); err != nil {
		return err
	}
	for index := range results {
		result := results[index]
		if err := ValidateCandidate(result.Artifact); err != nil {
			return fmt.Errorf("validate candidate %s: %w", result.Artifact.CandidateID, err)
		}
		goooPath := filepath.Join(caseDir, "candidates", result.Artifact.CandidateID+".gooo")
		irPath := filepath.Join(caseDir, "candidates", result.Artifact.CandidateID+".ir.json")
		if err := os.WriteFile(goooPath, []byte(RenderCandidateGooo(result.Artifact)), 0o644); err != nil {
			return err
		}
		if err := WriteJSON(irPath, result.Artifact); err != nil {
			return err
		}
		if _, err := CompileCandidate(goooPath, irPath); err != nil {
			return fmt.Errorf("compile candidate %s: %w", result.Artifact.CandidateID, err)
		}
		report.Candidates[index].ArtifactGooo = filepath.ToSlash(filepath.Join("cases", report.Scenario, "candidates", result.Artifact.CandidateID+".gooo"))
		report.Candidates[index].ArtifactIR = filepath.ToSlash(filepath.Join("cases", report.Scenario, "candidates", result.Artifact.CandidateID+".ir.json"))
	}
	return WriteJSON(filepath.Join(caseDir, "case-report.json"), report)
}

func BuildConformance(meta MetaContract, metaRaw []byte, reports []CaseReport) ConformanceReport {
	result := ConformanceReport{Schema: ConformanceSchema, Authority: meta.Authority, MetaDigest: Digest(metaRaw), Decision: DecisionClosed, Scenarios: len(reports), RepositoryWrites: 0, Cases: []ConformanceCase{}}
	for _, report := range reports {
		pass := report.Decision == report.ExpectedDecision
		result.Cases = append(result.Cases, ConformanceCase{Scenario: report.Scenario, Expected: report.ExpectedDecision, Observed: report.Decision, Pass: pass})
		switch report.Decision {
		case DecisionClosed:
			result.Closed++
		case DecisionUnknown:
			result.Unknown++
		case DecisionRefuted:
			result.Refuted++
		}
		if !pass {
			result.Decision = DecisionRefuted
		}
	}
	return result
}

func ValidateConformance(report ConformanceReport, meta MetaContract) error {
	if report.Schema != ConformanceSchema || report.Scenarios != meta.Denominator.Scenarios || len(report.Cases) != meta.Denominator.Scenarios || report.RepositoryWrites != 0 {
		return errors.New("conformance report does not match the .gooo denominator or input boundary")
	}
	for _, testCase := range report.Cases {
		if !testCase.Pass {
			return fmt.Errorf("fixed case %q observed %s, expected %s", testCase.Scenario, testCase.Observed, testCase.Expected)
		}
	}
	return nil
}

type fileSnapshot map[string]string

func SnapshotInput(root string) (fileSnapshot, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("input root must be a directory")
	}
	snapshot := fileSnapshot{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		snapshot[relative] = Digest(data)
		return nil
	})
	return snapshot, err
}

func SameSnapshot(before, after fileSnapshot) bool {
	if len(before) != len(after) {
		return false
	}
	for path, digest := range before {
		if after[path] != digest {
			return false
		}
	}
	return true
}

func PrepareOutput(outputDir, inputRoot string) error {
	if outputDir == "" || inputRoot == "" {
		return errors.New("caller-owned output and input root are required")
	}
	out, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	root, err := filepath.Abs(inputRoot)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, out)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return errors.New("caller-owned output must be outside the input repository")
	}
	if info, statErr := os.Stat(out); statErr == nil && !info.IsDir() {
		return errors.New("caller-owned output must be a directory")
	}
	if statErr := os.MkdirAll(out, 0o755); statErr != nil {
		return statErr
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("caller-owned output directory must be empty")
	}
	return nil
}

func sortedReports(reports []CaseReport) []CaseReport {
	result := append([]CaseReport(nil), reports...)
	sort.Slice(result, func(i, j int) bool { return result[i].Scenario < result[j].Scenario })
	return result
}
