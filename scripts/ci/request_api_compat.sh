#!/usr/bin/env bash
set -euo pipefail
test "$(git -C .request-diagnostic-baseline rev-parse HEAD)" = 790be850e98c06119330907a8fc2ae091e2bdc85
for file in request.go codec.go types.go request_test.go codec_test.go go.mod; do
  cmp "v3evidence/$file" ".request-diagnostic-baseline/v3evidence/$file"
done
cp scripts/ci/request_api_compat_test.go .request-diagnostic-baseline/v3evidence/zz_request_api_compat_test.go
cp scripts/ci/request_api_compat_test.go v3evidence/zz_request_api_compat_test.go
export REQUEST_API_SNAPSHOT
REQUEST_API_SNAPSHOT="${RUNNER_TEMP:?}/request-api-before.json"
(cd .request-diagnostic-baseline/v3evidence && go test -p 1 -tags=request_api_compat -count=1 -v -run '^TestRequestLegacyAPICompatibilitySnapshot$' ./...)
REQUEST_API_SNAPSHOT="${RUNNER_TEMP:?}/request-api-after.json"
(cd v3evidence && go test -p 1 -tags=request_api_compat -count=1 -v -run '^TestRequestLegacyAPICompatibilitySnapshot$' ./...)
cmp "${RUNNER_TEMP}/request-api-before.json" "${RUNNER_TEMP}/request-api-after.json"
git diff --exit-code -- go.mod go.sum v3evidence/go.mod v3evidence/go.sum
git -C .request-diagnostic-baseline diff --exit-code -- go.mod go.sum v3evidence/go.mod v3evidence/go.sum
