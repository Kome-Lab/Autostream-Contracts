package contracts

import (
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMutationGrantContractsBindExactRemoteSessionAndRejectUnknownFields(t *testing.T) {
	issue := readContractSchema(t, "update-agent-mutation-grant-issue-request.schema.json")
	issueFields := []string{
		"service_id", "lease_token", "lease_generation", "host_id", "target_id",
		"target_version", "deployment_mode", "operation", "plan_sha256", "session_id",
	}
	requireContractFields(t, issue.Required, issueFields...)
	if issue.AdditionalProperties != false || !issue.Properties["lease_token"].WriteOnly {
		t.Fatal("grant issue request must reject unknown fields and protect the lease token")
	}

	issued := readContractSchema(t, "update-agent-mutation-grant-issue-response.schema.json")
	requireContractFields(t, issued.Required, "grant_token", "expires_at")
	if issued.AdditionalProperties != false || !issued.Properties["grant_token"].WriteOnly ||
		issued.Properties["grant_token"].MinLength != 32 || issued.Properties["grant_token"].MaxLength != 256 {
		t.Fatal("issued mutation grant must be a bounded one-time secret")
	}
	if _, exists := issued.Properties["grant_id"]; exists {
		t.Fatal("public mutation grant response must not expose the internal grant ID")
	}

	consume := readContractSchema(t, "update-agent-mutation-grant-consume-request.schema.json")
	consumeFields := []string{
		"lease_generation", "host_id", "target_id", "target_version",
		"deployment_mode", "operation", "plan_sha256", "session_id",
	}
	requireContractFields(t, consume.Required, consumeFields...)
	if consume.AdditionalProperties != false {
		t.Fatal("grant consume request must reject unknown fields")
	}
	for _, forbidden := range []string{"job_id", "grant_token", "ssh_address", "ssh_user", "ssh_path", "path", "command"} {
		if _, exists := issue.Properties[forbidden]; exists {
			t.Fatalf("grant issue body exposes forbidden field %q", forbidden)
		}
		if _, exists := consume.Properties[forbidden]; exists {
			t.Fatalf("grant consume body exposes forbidden field %q", forbidden)
		}
	}
	for _, forbidden := range []string{"service_id", "lease_token"} {
		if _, exists := consume.Properties[forbidden]; exists {
			t.Fatalf("grant consume body exposes service-lease credential field %q", forbidden)
		}
	}
	for _, schema := range []contractSchemaNode{issue, consume} {
		if schema.Properties["plan_sha256"].Pattern != "^[a-f0-9]{64}$" {
			t.Fatal("mutation grant plan must use an exact lowercase SHA-256 binding")
		}
		if schema.Properties["session_id"].MinLength != 16 || schema.Properties["session_id"].MaxLength != 128 ||
			schema.Properties["session_id"].Pattern != "^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$" {
			t.Fatal("mutation grant session must be a bounded high-entropy identifier")
		}
		for _, operation := range []string{"apply", "reconcile"} {
			if !contractSliceHas(schema.Properties["operation"].Enum, operation) {
				t.Fatalf("mutation grant operation enum is missing %q", operation)
			}
		}
	}

	issueSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-issue-request.schema.json")
	consumeSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-consume-request.schema.json")
	responseSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-issue-response.schema.json")
	assertValidation := func(schema *jsonschema.Schema, body string, wantValid bool) {
		t.Helper()
		var instance any
		if err := json.Unmarshal([]byte(body), &instance); err != nil {
			t.Fatal(err)
		}
		err := schema.Validate(instance)
		if wantValid && err != nil {
			t.Fatalf("expected valid mutation grant contract for %s: %v", body, err)
		}
		if !wantValid && err == nil {
			t.Fatalf("expected mutation grant contract rejection for %s", body)
		}
	}
	hash := strings.Repeat("a", 64)
	lease := strings.Repeat("l", 43)
	grant := strings.Repeat("g", 43)
	validIssue := `{"service_id":"updater-1","lease_token":"` + lease + `","lease_generation":2,"host_id":"host-1","target_id":"worker-1","target_version":"v1.2.0","deployment_mode":"systemd","operation":"apply","plan_sha256":"` + hash + `","session_id":"session-contract-0001"}`
	validConsume := `{"lease_generation":2,"host_id":"host-1","target_id":"worker-1","target_version":"v1.2.0","deployment_mode":"systemd","operation":"apply","plan_sha256":"` + hash + `","session_id":"session-contract-0001"}`
	assertValidation(issueSchema, validIssue, true)
	assertValidation(consumeSchema, validConsume, true)
	assertValidation(responseSchema, `{"grant_token":"`+grant+`","expires_at":"2026-07-19T00:01:00Z"}`, true)
	assertValidation(issueSchema, strings.TrimSuffix(validIssue, "}")+`,"command":"systemctl restart x"}`, false)
	assertValidation(consumeSchema, strings.TrimSuffix(validConsume, "}")+`,"grant_token":"`+grant+`"}`, false)
	assertValidation(consumeSchema, strings.Replace(validConsume, hash, strings.ToUpper(hash), 1), false)
	assertValidation(consumeSchema, strings.Replace(validConsume, `"operation":"apply"`, `"operation":"probe"`, 1), false)
	assertValidation(consumeSchema, strings.Replace(validConsume, "session-contract-0001", "short", 1), false)
	assertValidation(responseSchema, `{"grant_token":"`+grant+`","expires_at":"2026-07-19T00:01:00Z","session_id":"session-contract-0001"}`, false)

	now := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC)
	issueBody, err := json.Marshal(UpdateAgentMutationGrantIssueRequest{
		ServiceID: "updater-1", LeaseToken: lease, LeaseGeneration: 2, ExecutionHostID: "host-1",
		TargetID: "worker-1", TargetVersion: "v1.2.0", DeploymentMode: SystemUpdateDeploymentSystemd,
		Operation: SystemUpdateMutationApply, PlanSHA256: hash, SessionID: "session-contract-0001",
	})
	if err != nil {
		t.Fatal(err)
	}
	consumeBody, err := json.Marshal(UpdateAgentMutationGrantConsumeRequest{
		LeaseGeneration: 2, ExecutionHostID: "host-1", TargetID: "worker-1",
		TargetVersion: "v1.2.0", DeploymentMode: SystemUpdateDeploymentSystemd,
		Operation: SystemUpdateMutationApply, PlanSHA256: hash, SessionID: "session-contract-0001",
	})
	if err != nil {
		t.Fatal(err)
	}
	responseBody, err := json.Marshal(UpdateAgentMutationGrantIssueResponse{GrantToken: grant, ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	for schema, body := range map[*jsonschema.Schema][]byte{
		issueSchema: issueBody, consumeSchema: consumeBody, responseSchema: responseBody,
	} {
		var instance any
		if err := json.Unmarshal(body, &instance); err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(instance); err != nil {
			t.Fatalf("Go mutation grant JSON violates schema: %v body=%s", err, body)
		}
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"/services/update-jobs/{id}/mutation-grants:",
		"/services/update-jobs/{id}/mutation-grants/consume:",
		"UpdateAgentMutationGrantIssueRequest:",
		"UpdateAgentMutationGrantIssueResponse:",
		"UpdateAgentMutationGrantConsumeRequest:",
		`"201":`, "Always no-store because the response contains a one-time grant credential.",
		"supplied as the Bearer credential", "exact same grant, session, and binding", "different reuse returns 409",
	} {
		if !strings.Contains(string(openAPI), marker) {
			t.Fatalf("control OpenAPI is missing mutation-grant marker %q", marker)
		}
	}
	if strings.Contains(string(openAPI), "grant_id:") {
		t.Fatal("control OpenAPI exposes the internal mutation grant ID")
	}
}

func TestUpdaterRegistrationScopesHeartbeatAndEmailHTMLContracts(t *testing.T) {
	if ServiceUpdateAgent != "update_agent" || ServiceStatusUpdating != "updating" {
		t.Fatal("update agent service type or updating status changed")
	}
	if ScopeUpdatesClaim != "updates.claim" || ScopeUpdatesReport != "updates.report" || ScopeUpdatesAuthorize != "updates.authorize" {
		t.Fatal("update agent scopes changed")
	}
	if SystemUpdateReconciling != "reconciling" ||
		SystemUpdateDeploymentSystemd != "systemd" || SystemUpdateDeploymentDocker != "docker" {
		t.Fatal("system update recovery or deployment values changed")
	}

	heartbeat := readContractSchema(t, "heartbeat.schema.json")
	if !contractSliceHas(heartbeat.Properties["status"].Enum, "updating") {
		t.Fatal("heartbeat status enum is missing updating")
	}
	for _, name := range []string{"service-token-create-request.schema.json", "service-token.schema.json"} {
		token := readContractSchema(t, name)
		if !contractSliceHas(token.Properties["service_type"].Enum, "update_agent") {
			t.Fatalf("%s is missing update_agent", name)
		}
		for _, scope := range []string{"updates.claim", "updates.report", "updates.authorize"} {
			if !contractSliceHas(token.Properties["scopes"].Items.Enum, scope) {
				t.Fatalf("%s is missing %s", name, scope)
			}
		}
	}
	issuedTokenRequest, err := json.Marshal(ServiceTokenCreateRequest{
		ServiceType: ServiceUpdateAgent,
		Scopes:      []ServiceScope{ScopeServiceRegister, ScopeServiceHeartbeat, ScopeServiceConfigRead, ScopeServiceLogsWrite, ScopeServiceStatusWrite, ScopeUpdatesClaim, ScopeUpdatesReport, ScopeUpdatesAuthorize},
	})
	if err != nil {
		t.Fatal(err)
	}
	var issuedTokenInstance any
	if err := json.Unmarshal(issuedTokenRequest, &issuedTokenInstance); err != nil {
		t.Fatal(err)
	}
	if err := compileContractJSONSchema(t, "service-token-create-request.schema.json", "encoder-output-relay-capabilities.schema.json").Validate(issuedTokenInstance); err != nil {
		t.Fatalf("Control Panel-issued update_agent token violates the contract: %v body=%s", err, issuedTokenRequest)
	}

	email := readContractSchema(t, "service-notification-email-request.schema.json")
	requireContractFields(t, email.Required, "recipients", "subject", "text")
	if contractSliceHas(email.Required, "html") {
		t.Fatal("email html must remain optional")
	}
	html := email.Properties["html"]
	if html.MaxLength != 65536 || html.Pattern != "^[^\\u0000]*$" {
		t.Fatalf("unexpected email html limits: %#v", html)
	}
	body, err := json.Marshal(ServiceNotificationEmailRequest{
		Recipients: []string{"ops@example.jp"}, Subject: "更新", Text: "plain fallback",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), `"html"`) {
		t.Fatalf("empty optional html must be omitted: %s", body)
	}
	openAPIBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"optional HTML alternative", "html:", "maxLength: 65536"} {
		if !strings.Contains(string(openAPIBody), marker) {
			t.Fatalf("control OpenAPI is missing email HTML marker %q", marker)
		}
	}
}
