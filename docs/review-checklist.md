# Semantic review checklist

The pull request is ready for semantic review when the CI artifact contains:

- one `.gooo` candidate, typed IR JSON, caller-owned patch, and dossier for each of the three declared operators in every fixed cell;
- exactly twelve cells and `4 CLOSED`, `4 UNKNOWN`, and `4 REFUTED` decisions, with all twelve expected decisions passing;
- `4/4/4` proof counts and `4/4/4` indicator counts, plus the exact paired before/after indicator vector;
- six-field UNKNOWN records for missing identity and unavailable external utility evidence;
- REFUTED dossiers for reason/effect drift, hidden-only repair, and normal/regression corpus changes;
- `repository_writes=0`, `local_test_executions=0`, `cross_project_required_gates=0`, and integer root-excluded inventory/test metrics.

The release reviewer should compare the generated candidate and paired evaluator digests with the CI artifact, confirm that the optional v0.1.1 oracle pins remain non-required, verify the preserved release ID/assets, and verify that no source mutation or Git integration is performed by the command.
