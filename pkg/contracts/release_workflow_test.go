package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractsReleaseWorkflowIsImmutableAndProducesPortableChecksums(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "release-contracts.yml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)

	for _, want := range []string{
		"INPUT_VERSION: ${{ inputs.version }}",
		"INPUT_PUSH_RELEASE: ${{ inputs.push_release }}",
		"group: release-${{ github.repository }}-${{ github.ref_type == 'tag' && github.ref_name || inputs.version }}",
		"cancel-in-progress: false",
		"attestations: write",
		"id-token: write",
		`version="${INPUT_VERSION}"`,
		`push_release="${INPUT_PUSH_RELEASE}"`,
		"find . -type f ! -path './checksums.txt'",
		`(cd artifacts && sha256sum --text "${artifact}.tar.gz" > "${artifact}.tar.gz.sha256")`,
		`(cd artifacts && sha256sum --check --strict "${artifact}.tar.gz.sha256")`,
		"- name: Attest release assets",
		"uses: actions/attest@",
		"subject-path: artifacts/*",
		"- name: Require repository immutable releases",
		`"repos/${GITHUB_REPOSITORY}/immutable-releases"`,
		"(.enabled == true)",
		`gh api --paginate "repos/${GITHUB_REPOSITORY}/releases?per_page=100"`,
		`select(.tag_name == $tag)`,
		`git/ref/tags/${RELEASE_VERSION}`,
		`"${ref_sha}" != "${GITHUB_SHA}"`,
		"workflow staging namespace",
		"workflow_dispatch may not overwrite or reuse it",
		"failed or partial release requires a new version",
		"id: create-draft",
		`draft_tag="${RELEASE_VERSION}-staging-${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}"`,
		`contracts-release-body.md`,
		`contracts-release-body.sha256`,
		"AutoStream Contracts ${RELEASE_VERSION}",
		`--method POST "repos/${GITHUB_REPOSITORY}/releases"`,
		`-f tag_name="${DRAFT_TAG}"`,
		`-f target_commitish="${GITHUB_SHA}"`,
		`-f body="$(< "${release_body_path}")"`,
		"-F draft=true",
		"https://uploads.github.com/repos/${GITHUB_REPOSITORY}/releases/${DRAFT_RELEASE_ID}/assets?name=${name}",
		"name: Publish verified release atomically",
		"DRAFT_RELEASE_ID: ${{ steps.create-draft.outputs.release_id }}",
		`(.draft == true)`,
		`jq -j '.body'`,
		`cmp -s "${release_body_path}" "${actual_body_path}"`,
		`immutable-release-settings-prepublish.json`,
		"appeared during staging; refusing to overwrite it",
		"moved during staging; refusing to publish mismatched assets",
		`final_draft_json="${RUNNER_TEMP}/contracts-final-draft-release.json"`,
		"appeared immediately before publication; refusing to overwrite it",
		`--method POST "repos/${GITHUB_REPOSITORY}/git/refs"`,
		`-f ref="refs/tags/${RELEASE_VERSION}"`,
		`-f sha="${GITHUB_SHA}"`,
		`"${RUNNER_TEMP}/contracts-owned-final-tag"`,
		"Could not atomically claim tag",
		"does not resolve to workflow commit ${GITHUB_SHA} immediately before publish",
		`gh api --method DELETE "repos/${GITHUB_REPOSITORY}/git/refs/tags/${DRAFT_TAG}"`,
		`--method PATCH "repos/${GITHUB_REPOSITORY}/releases/${DRAFT_RELEASE_ID}"`,
		`-f tag_name="${RELEASE_VERSION}"`,
		`-f target_commitish="${GITHUB_SHA}"`,
		"-F draft=false",
		`.draft == false`,
		`.immutable == true`,
		`expected_archive="autostream-contracts_${RELEASE_VERSION}.tar.gz"`,
		`(.assets | length == 2)`,
		`[.assets[] | {name, size, digest}] | sort_by(.name)`,
		`gh attestation verify "${asset_path}" --repo "${GITHUB_REPOSITORY}"`,
		"name: Preserve failed release state for manual recovery",
		`if: ${{ always() && steps.create-draft.outputs.release_id != '' }}`,
		`if [[ "${is_draft}" == "true" && "${release_tag}" == "${DRAFT_TAG}" ]]; then`,
		`elif [[ "${release_tag}" == "${RELEASE_VERSION}" ]]; then`,
		"all refs for manual recovery; no release or ref was deleted",
		"published-but-unverified",
		"diff -u",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("release-contracts.yml is missing release-safety marker %q", want)
		}
	}

	if count := strings.Count(raw, "${{ inputs."); count != 2 {
		t.Fatalf("direct workflow dispatch input expressions must appear only in step env declarations, found %d occurrences", count)
	}
	for _, forbidden := range []string{
		`version="${{ inputs.version }}"`,
		`push_release="${{ inputs.push_release }}"`,
		"find . -type f -print0",
		`sha256sum "artifacts/${artifact}.tar.gz"`,
		"continue-on-error: true",
		"softprops/action-gh-release",
		"generate_release_notes:",
		`gh api --method DELETE "repos/${GITHUB_REPOSITORY}/releases/${DRAFT_RELEASE_ID}"`,
		`gh api --method DELETE "repos/${GITHUB_REPOSITORY}/git/refs/tags/${RELEASE_VERSION}"`,
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("release-contracts.yml contains unsafe or non-portable marker %q", forbidden)
		}
	}

	createDraft := strings.Index(raw, `--method POST "repos/${GITHUB_REPOSITORY}/releases"`)
	attest := strings.Index(raw, "- name: Attest release assets")
	prepublishCheck := strings.Index(raw, "appeared during staging; refusing to overwrite it")
	finalNamespaceCheck := strings.LastIndex(raw, "appeared immediately before publication; refusing to overwrite it")
	finalClaim := strings.Index(raw, `--method POST "repos/${GITHUB_REPOSITORY}/git/refs"`)
	finalTagCheck := strings.LastIndex(raw, "does not resolve to workflow commit ${GITHUB_SHA} immediately before publish")
	publish := strings.Index(raw, `--method PATCH "repos/${GITHUB_REPOSITORY}/releases/${DRAFT_RELEASE_ID}"`)
	cleanup := strings.Index(raw, "- name: Preserve failed release state for manual recovery")
	if !(createDraft >= 0 &&
		attest > createDraft &&
		prepublishCheck > attest &&
		finalNamespaceCheck > prepublishCheck &&
		finalClaim > finalNamespaceCheck &&
		finalTagCheck > finalClaim &&
		publish > finalTagCheck &&
		cleanup > publish) {
		t.Fatalf("release steps are not ordered as stage, attest, final namespace check, atomic tag claim, exact tag recheck, publish, cleanup")
	}
	publishedTagCheck := strings.LastIndex(raw, "Published tag ${RELEASE_VERSION} does not resolve to workflow commit ${GITHUB_SHA}")
	stagingTagDelete := strings.Index(raw, `gh api --method DELETE "repos/${GITHUB_REPOSITORY}/git/refs/tags/${DRAFT_TAG}"`)
	if !(publishedTagCheck > publish && stagingTagDelete > publishedTagCheck && cleanup > stagingTagDelete) {
		t.Fatalf("workflow-owned staging tag may be deleted only after successful published release and final-tag verification")
	}
	if strings.Count(raw, `gh api --method DELETE "repos/${GITHUB_REPOSITORY}/git/refs/tags/${DRAFT_TAG}"`) != 1 {
		t.Fatalf("workflow-owned staging tag must have exactly one success-only deletion")
	}
	if strings.Count(raw, `--method POST "repos/${GITHUB_REPOSITORY}/git/refs"`) != 1 {
		t.Fatalf("workflow_dispatch final tag must be atomically claimed exactly once")
	}
	if strings.Count(raw, `cmp -s "${release_body_path}"`) != 4 {
		t.Fatalf("deterministic release body must be compared at draft, prepublish, final-draft, and published checkpoints")
	}
}
