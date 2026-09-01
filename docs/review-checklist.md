# Semantic review checklist

The pull request is ready for semantic review when the CI artifact contains:

- one `.gooo` candidate and one typed IR JSON artifact for each of the three declared operators in every fixed case;
- `4 CLOSED`, `2 UNKNOWN`, and `1 REFUTED` case decisions, with all seven expected decisions passing;
- a six-field UNKNOWN record for the ambiguous and missing-origin cases;
- a reason-drift REFUTED candidate whose replay still exposes the counterexample;
- `repository_writes=0` and integer inventory/test metrics.

The release reviewer should compare the generated candidate digests with the CI artifact, confirm that the optional v0.1.1 oracle pins remain non-required, and verify that no source mutation or Git integration is performed by the command.
