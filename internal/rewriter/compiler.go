package rewriter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type CandidateSource struct {
	CandidateID         string
	Scenario            string
	Operator            string
	Status              string
	SourceDigest        string
	OriginSourceDigest  string
	ToolchainDigest     string
	InputIRDigest       string
	TransformedIRDigest string
}

// CompileCandidate is the small fixture compiler: it parses the generated
// .gooo candidate, loads its typed IR artifact, and verifies that the source
// declaration and JSON artifact describe the same candidate.
func CompileCandidate(goooPath, irPath string) (CandidateArtifact, error) {
	source, err := parseCandidateSource(goooPath)
	if err != nil {
		return CandidateArtifact{}, err
	}
	raw, err := os.ReadFile(irPath)
	if err != nil {
		return CandidateArtifact{}, err
	}
	var artifact CandidateArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return CandidateArtifact{}, fmt.Errorf("parse candidate IR: %w", err)
	}
	if err := artifact.Validate(); err != nil {
		return CandidateArtifact{}, err
	}
	if source.CandidateID != artifact.CandidateID || source.Scenario != artifact.Scenario || source.Operator != artifact.Operator || source.Status != artifact.CandidateStatus ||
		source.SourceDigest != artifact.SourceDigest || source.OriginSourceDigest != artifact.OriginSourceDigest || source.ToolchainDigest != artifact.ToolchainDigest ||
		source.InputIRDigest != artifact.InputIRDigest || source.TransformedIRDigest != artifact.TransformedIRDigest {
		return CandidateArtifact{}, errors.New("candidate .gooo and typed IR artifact disagree")
	}
	if artifact.CandidateStatus != DecisionUnknown {
		if err := artifact.IR.Validate(); err != nil {
			return CandidateArtifact{}, fmt.Errorf("compile typed IR: %w", err)
		}
	}
	return artifact, nil
}

func parseCandidateSource(path string) (CandidateSource, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return CandidateSource{}, err
	}
	var source CandidateSource
	for lineNumber, rawLine := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens, err := tokenize(line)
		if err != nil {
			return CandidateSource{}, fmt.Errorf("candidate line %d: %w", lineNumber+1, err)
		}
		if len(tokens) == 0 {
			continue
		}
		switch {
		case tokens[0] == "gooo":
			if len(tokens) != 3 || tokens[1] != "candidate_rewrite" || tokens[2] != "v1" {
				return CandidateSource{}, errors.New("invalid candidate header")
			}
		case tokens[0] == "candidate":
			values := looseKeyValues(tokens[1:])
			source.CandidateID, source.Scenario, source.Operator, source.Status = values["id"], values["scenario"], values["operator"], values["status"]
		case strings.HasPrefix(tokens[0], "source_digest="):
			values := looseKeyValues(tokens)
			source.SourceDigest, source.OriginSourceDigest, source.ToolchainDigest = values["source_digest"], values["origin_source_digest"], values["toolchain_digest"]
		case strings.HasPrefix(tokens[0], "input_ir_digest="):
			values := looseKeyValues(tokens)
			source.InputIRDigest, source.TransformedIRDigest = values["input_ir_digest"], values["transformed_ir_digest"]
		}
	}
	if source.CandidateID == "" || source.Scenario == "" || source.Operator == "" || !allowedDecision(source.Status) {
		return CandidateSource{}, errors.New("candidate source declaration is incomplete")
	}
	return source, nil
}

func looseKeyValues(tokens []string) map[string]string {
	values := map[string]string{}
	for _, token := range tokens {
		key, value, ok := strings.Cut(token, "=")
		if ok && key != "" {
			values[key] = value
		}
	}
	return values
}

func ValidateCandidate(candidate CandidateArtifact) error {
	if err := candidate.Validate(); err != nil {
		return err
	}
	if candidate.CandidateStatus == DecisionUnknown {
		return nil
	}
	if err := candidate.IR.Validate(); err != nil {
		return err
	}
	if candidate.TransformedIRDigest != candidate.IR.CanonicalDigest() {
		return errors.New("candidate IR digest does not recompute")
	}
	return nil
}
