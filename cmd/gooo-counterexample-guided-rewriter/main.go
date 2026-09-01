package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-counterexample-guided-rewriter/internal/rewriter"
)

func main() {
	metaPath := flag.String("meta", ".gooo/rewrite.gooo", "authoritative .gooo contract")
	inputPath := flag.String("input", "", "reducer-style counterexample JSON")
	inputRoot := flag.String("input-root", ".", "immutable input repository root")
	outputDir := flag.String("output", "", "empty caller-owned output directory outside input-root")
	all := flag.Bool("fixed", false, "execute every fixed scenario declared in .gooo")
	flag.Parse()

	if *outputDir == "" || (*inputPath == "" && !*all) || (*inputPath != "" && *all) {
		fail("provide exactly one of -input or -fixed and provide -output")
	}
	meta, metaRaw, err := rewriter.LoadMeta(*metaPath)
	if err != nil {
		fail(err.Error())
	}
	if err := rewriter.PrepareOutput(*outputDir, *inputRoot); err != nil {
		fail(err.Error())
	}
	before, err := rewriter.SnapshotInput(*inputRoot)
	if err != nil {
		fail(err.Error())
	}

	if *inputPath != "" {
		if err := runOne(meta, *inputPath, *outputDir); err != nil {
			fail(err.Error())
		}
	} else {
		if err := runFixed(meta, metaRaw, *inputRoot, *outputDir); err != nil {
			fail(err.Error())
		}
	}

	after, err := rewriter.SnapshotInput(*inputRoot)
	if err != nil {
		fail(err.Error())
	}
	if !rewriter.SameSnapshot(before, after) {
		fail("repository_writes=1: input repository changed during candidate generation")
	}
	fmt.Println("repository_writes=0")
}

func runOne(meta rewriter.MetaContract, inputPath, outputDir string) error {
	counterexample, _, err := rewriter.LoadCounterexample(inputPath)
	if err != nil {
		return err
	}
	report, results, err := rewriter.GenerateCase(meta, counterexample)
	if err != nil {
		return err
	}
	report.RepositoryWrites = 0
	if err := rewriter.WriteCaseArtifacts(outputDir, report, results); err != nil {
		return err
	}
	fmt.Printf("scenario=%s decision=%s candidates=%d\n", report.Scenario, report.Decision, len(report.Candidates))
	return nil
}

func runFixed(meta rewriter.MetaContract, metaRaw []byte, inputRoot, outputDir string) error {
	reports := make([]rewriter.CaseReport, 0, len(meta.Scenarios))
	for _, scenario := range meta.Scenarios {
		counterexample, _, err := rewriter.LoadCounterexample(filepath.Join(inputRoot, scenario.Fixture))
		if err != nil {
			return fmt.Errorf("%s: %w", scenario.ID, err)
		}
		report, results, err := rewriter.GenerateCase(meta, counterexample)
		if err != nil {
			return fmt.Errorf("%s: %w", scenario.ID, err)
		}
		report.RepositoryWrites = 0
		if err := rewriter.WriteCaseArtifacts(outputDir, report, results); err != nil {
			return fmt.Errorf("%s: %w", scenario.ID, err)
		}
		reports = append(reports, report)
	}
	conformance := rewriter.BuildConformance(meta, metaRaw, reports)
	if err := rewriter.ValidateConformance(conformance, meta); err != nil {
		return err
	}
	if err := rewriter.WriteJSON(filepath.Join(outputDir, "conformance-report.json"), conformance); err != nil {
		return err
	}
	fmt.Printf("scenarios=%d closed=%d unknown=%d refuted=%d\n", conformance.Scenarios, conformance.Closed, conformance.Unknown, conformance.Refuted)
	return nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
