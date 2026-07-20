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
		`version="${INPUT_VERSION}"`,
		`push_release="${INPUT_PUSH_RELEASE}"`,
		"find . -type f ! -path './checksums.txt'",
		`(cd artifacts && sha256sum --text "${artifact}.tar.gz" > "${artifact}.tar.gz.sha256")`,
		`(cd artifacts && sha256sum --check --strict "${artifact}.tar.gz.sha256")`,
		`gh api --paginate "repos/${GITHUB_REPOSITORY}/releases?per_page=100"`,
		`select(.tag_name == $tag)`,
		`git/ref/tags/${RELEASE_VERSION}`,
		`--method POST "repos/${GITHUB_REPOSITORY}/git/refs"`,
		`-f ref="refs/tags/${RELEASE_VERSION}"`,
		`-f sha="${GITHUB_SHA}"`,
		`"${ref_sha}" != "${GITHUB_SHA}"`,
		"already exists (including drafts)",
		"workflow_dispatch may not overwrite or reuse it",
		"failed or partial release requires a new version",
		"target_commitish: ${{ github.sha }}",
		"fail_on_unmatched_files: true",
		"overwrite_files: false",
		"name: Verify published release",
		`.draft == false`,
		`expected_archive="autostream-contracts_${RELEASE_VERSION}.tar.gz"`,
		`(.assets | length == 2)`,
		`[.assets[] | {name, size, digest}] | sort_by(.name)`,
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
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("release-contracts.yml contains unsafe or non-portable marker %q", forbidden)
		}
	}
}
