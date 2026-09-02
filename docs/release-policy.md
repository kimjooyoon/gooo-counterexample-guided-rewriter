# Release policy

The repository immutable-releases setting is enabled through GitHub's repository administration API. The setting guard fails closed when the endpoint is unavailable, the response is missing `enabled`, or the value is not `true`. The release guard fails closed when the published Release API response is missing `immutable` or does not report `true`.

v0.1.0 predates the repository setting and is intentionally preserved as-is with its original `immutable=false` API value. Immutable release `v0.1.1` (release ID `380240322`) is preserved with `gooo-counterexample-guided-rewriter-v0.1.1.tar.gz` (`sha256:2019d42b7050f2c894f839d9c87c261291b43bd54eb7251e11e8485059b907cf`) and its `.sha256` sidecar (`sha256:a12ef67f3b6ffcae9b33cb26072fe60d70a6a37dd35712a01279e634248187b4`). New changes follow a PR-first green path; the default branch preserves prior evidence, and the next release is draft-first before immutable publication. Public release deletion, retagging, overwriting, and asset replacement are forbidden.

All generation and before/after validation run in pull-request and default-branch Actions. The runtime reports zero repository writes, zero local test executions, and zero required cross-project gates. Its only output is the caller-owned output directory.
