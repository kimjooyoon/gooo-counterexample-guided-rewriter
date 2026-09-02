package rewriter

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestFixedCorpusDecisions(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	meta, raw, err := LoadMeta(filepath.Join(root, ".gooo", "rewrite.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	reports := make([]CaseReport, 0, len(meta.Scenarios))
	for _, scenario := range meta.Scenarios {
		counterexample, _, err := LoadCounterexample(filepath.Join(root, scenario.Fixture))
		if err != nil {
			t.Fatalf("%s: %v", scenario.ID, err)
		}
		report, results, err := GenerateCase(meta, counterexample)
		if err != nil {
			t.Fatalf("%s: %v", scenario.ID, err)
		}
		if len(results) != 3 {
			t.Fatalf("%s: got %d candidates", scenario.ID, len(results))
		}
		if report.Decision != scenario.Expected {
			t.Fatalf("%s: got %s, want %s", scenario.ID, report.Decision, scenario.Expected)
		}
		reports = append(reports, report)
	}
	conformance := BuildConformance(meta, raw, reports)
	if err := ValidateConformance(conformance, meta); err != nil {
		t.Fatal(err)
	}
	if conformance.Closed != 4 || conformance.Unknown != 4 || conformance.Refuted != 4 {
		t.Fatalf("unexpected counts: %+v", conformance)
	}
}

func TestExplicitFailClosedBranchIsNotFixedPoint(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	meta, _, err := LoadMeta(filepath.Join(root, ".gooo", "rewrite.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	counterexample, _, err := LoadCounterexample(filepath.Join(root, "fixtures", "cases", "closed-branch-split.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, results, err := GenerateCase(meta, counterexample)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Accepted {
			if result.Artifact.IR.Terminal.Decision != DecisionClosed || result.Artifact.IR.Terminal.CounterexampleVisible {
				t.Fatalf("accepted branch did not explicitly close and remove the causal input: %+v", result.Artifact.IR.Terminal)
			}
			foundExplicit := false
			for _, node := range result.Artifact.IR.Nodes {
				if node.ID == "branch-result" && attributes(node)["decision_branch"] == "explicit_fail_closed" {
					foundExplicit = true
				}
			}
			if !foundExplicit {
				t.Fatal("accepted branch did not declare an explicit fail-closed branch")
			}
		}
	}
}

func TestReasonAndEffectDriftAreRefuted(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	meta, _, err := LoadMeta(filepath.Join(root, ".gooo", "rewrite.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	counterexample, _, err := LoadCounterexample(filepath.Join(root, "fixtures", "cases", "reason-drift.json"))
	if err != nil {
		t.Fatal(err)
	}
	report, results, err := GenerateCase(meta, counterexample)
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionRefuted {
		t.Fatalf("got %s, want REFUTED", report.Decision)
	}
	for _, result := range results {
		if result.Artifact.Operator == "reason-preserving-branch-split" && !result.Refuted {
			t.Fatal("reason drift candidate was not refuted")
		}
	}
}

func TestOutputBoundaryRejectsInputRoot(t *testing.T) {
	if err := PrepareOutput(".", "."); err == nil {
		t.Fatal("input root was accepted as output")
	}
}
