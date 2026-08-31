package contracts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const officialCIWorkflowTestMutantEnvironment = "AUTOSTREAM_CI_WORKFLOW_TEST_MUTANT"

func TestOfficialOpenAPICIBlocksOnCanonicalAllContractVerification(t *testing.T) {
	workflowBody, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := strings.ReplaceAll(string(workflowBody), "\r\n", "\n")
	applyOfficialCIWorkflowTestMutant(t, &workflow, nil, os.Getenv(officialCIWorkflowTestMutantEnvironment))
	openAPIJob := workflowJobSection(t, workflow, "openapi")
	for _, required := range []string{
		"runs-on: ubuntu-24.04",
		"timeout-minutes: 10",
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1",
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
		"go-version-file: go.mod",
		"actions/setup-node@820762786026740c76f36085b0efc47a31fe5020",
		"name: Verify all OpenAPI contracts and characterizations",
		"run: node scripts/characterize-contracts.mjs verify",
		"name: Lint Discord Bot OpenAPI",
		"@redocly/cli@2.39.0 lint",
		"openapi/discord-bot-api.yaml",
		"name: Bundle and verify Discord Bot OpenAPI",
		"@redocly/cli@2.39.0 bundle",
		"discord-bot-start-job-request.schema",
		"18446744073709551615|18446744073709552000",
	} {
		if !strings.Contains(openAPIJob, required) {
			t.Fatalf("Official openapi job is missing blocking marker %q", required)
		}
	}
	for _, forbidden := range []string{"continue-on-error:", "|| true"} {
		if strings.Contains(openAPIJob, forbidden) {
			t.Fatalf("Official openapi job swallows failure with %q", forbidden)
		}
	}

	scriptBody, err := os.ReadFile(filepath.Join("..", "..", "scripts", "characterize-contracts.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(scriptBody)
	applyOfficialCIWorkflowTestMutant(t, nil, &script, os.Getenv(officialCIWorkflowTestMutantEnvironment))
	entrypoints := javascriptArraySection(t, script, "API_ENTRYPOINTS")
	for _, required := range []string{
		"\"openapi/control-api.yaml\"",
		"\"openapi/discord-bot-api.yaml\"",
		"\"openapi/encoder-recorder-api.yaml\"",
		"\"openapi/observability-api.yaml\"",
	} {
		if !strings.Contains(entrypoints, required) {
			t.Fatalf("canonical OpenAPI entrypoint inventory is missing %q", required)
		}
	}
	for _, required := range []string{
		"const REDOCLY_PACKAGE = \"@redocly/cli@2.39.0\";",
		"const REDOCLY_VERSION = \"2.39.0\";",
		"compareDirectoriesExactly(COMMITTED_ROOT, snapshotRoot)",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("canonical characterization script is missing %q", required)
		}
	}
}

func TestOfficialOpenAPICIMutationSensitivity(t *testing.T) {
	for _, test := range []struct {
		name        string
		wantFailure string
	}{
		{name: "control_api_omitted", wantFailure: "openapi/control-api.yaml"},
		{name: "characterization_verify_removed", wantFailure: "run: node scripts/characterize-contracts.mjs verify"},
	} {
		t.Run(test.name, func(t *testing.T) {
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(executable,
				"-test.run=^TestOfficialOpenAPICIBlocksOnCanonicalAllContractVerification$",
				"-test.count=1",
			)
			command.Env = append(os.Environ(), officialCIWorkflowTestMutantEnvironment+"="+test.name)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("workflow mutant %s was not rejected:\n%s", test.name, output)
			}
			if !strings.Contains(string(output), test.wantFailure) {
				t.Fatalf("workflow mutant %s failed for the wrong reason; wanted %q:\n%s",
					test.name, test.wantFailure, output)
			}
		})
	}
}

func applyOfficialCIWorkflowTestMutant(
	t *testing.T,
	workflow *string,
	script *string,
	mutant string,
) {
	t.Helper()
	switch mutant {
	case "":
	case "control_api_omitted":
		if script != nil {
			*script = strings.Replace(*script,
				"\"openapi/control-api.yaml\"", "\"openapi/control-api-omitted.yaml\"", 1)
		}
	case "characterization_verify_removed":
		if workflow != nil {
			*workflow = strings.Replace(*workflow,
				"node scripts/characterize-contracts.mjs verify",
				"node scripts/characterize-contracts.mjs removed", 1)
		}
	default:
		t.Fatalf("unknown Official CI workflow test mutant %q", mutant)
	}
}

func javascriptArraySection(t *testing.T, script string, name string) string {
	t.Helper()
	startMarker := "const " + name + " = ["
	start := strings.Index(script, startMarker)
	if start < 0 {
		t.Fatalf("JavaScript array %s is missing", name)
	}
	section := script[start+len(startMarker):]
	end := strings.Index(section, "];")
	if end < 0 {
		t.Fatalf("JavaScript array %s is unterminated", name)
	}
	return section[:end]
}

func workflowJobSection(t *testing.T, workflow string, name string) string {
	t.Helper()
	startMarker := "  " + name + ":\n"
	start := strings.Index(workflow, startMarker)
	if start < 0 {
		t.Fatalf("workflow job %q is missing", name)
	}
	section := workflow[start+len(startMarker):]
	lines := strings.Split(section, "\n")
	end := len(lines)
	for index, line := range lines {
		if strings.HasPrefix(line, "  ") &&
			!strings.HasPrefix(line, "    ") &&
			strings.HasSuffix(line, ":") {
			end = index
			break
		}
	}
	return strings.Join(lines[:end], "\n")
}
