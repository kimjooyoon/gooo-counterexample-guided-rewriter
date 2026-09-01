# Release policy

The repository immutable-releases setting is enabled through GitHub's repository administration API. The setting guard fails closed when the endpoint is unavailable, the response is missing `enabled`, or the value is not `true`. The release guard fails closed when the published Release API response is missing `immutable` or does not report `true`.

v0.1.0 predates the repository setting and is intentionally preserved as-is with its original `immutable=false` API value. New releases are created with all assets present at publication time and are verified through the Release API before the workflow succeeds.
