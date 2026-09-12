package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNotificationChannelSchemasDocumentEmailSecretBoundary(t *testing.T) {
	writeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-channel-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	publicBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-channel.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	observabilityOpenAPIBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "observability-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\"email\"",
		"\"webhook_url\"",
		"\"email_recipients\"",
		"\"uses_global_smtp\"",
		"\"format\": \"email\"",
		"\"const\": true",
		"\"writeOnly\": true",
		"Write-only Discord, Slack, or generic webhook URL",
	} {
		if !strings.Contains(string(writeBody), want) {
			t.Fatalf("notification-channel-write.schema.json is missing email marker %q", want)
		}
	}
	for _, want := range []string{
		"\"uses_global_smtp\"",
		"\"masked_email_target\"",
	} {
		if !strings.Contains(string(publicBody), want) {
			t.Fatalf("notification-channel.schema.json is missing public email marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"\"smtp_host\"",
		"\"smtp_from\"",
		"\"smtp_username\"",
		"\"smtp_password\"",
		"\"email_recipients\"",
	} {
		if strings.Contains(string(publicBody), forbidden) {
			t.Fatalf("notification-channel.schema.json exposes raw email notification field %q", forbidden)
		}
	}
	for _, want := range []string{
		"NotificationChannel:",
		"uses_global_smtp:",
		"proxied notification channels without raw webhook URLs or raw SMTP settings",
		"masked_email_target:",
		"writeOnly: true",
		"format: email",
		"Raw email recipients and SMTP settings are never returned as unmasked response fields.",
	} {
		if !strings.Contains(string(openapiBody), want) {
			t.Fatalf("control-api.yaml is missing email notification marker %q", want)
		}
	}
	controlNotification := requireControlNotificationComponentSection(t, string(openapiBody), "NotificationChannel:")
	observabilityNotification := requireControlNotificationComponentSection(t, string(observabilityOpenAPIBody), "NotificationChannel:")
	for _, removed := range []string{"smtp_password_configured:", "Deprecated direct-Observability SMTP compatibility status."} {
		if strings.Contains(controlNotification, removed) || strings.Contains(observabilityNotification, removed) {
			t.Fatalf("notification-channel OpenAPI retained removed SMTP field %q", removed)
		}
	}
	requireControlOpenAPINotificationEmail(t)
	for _, want := range []string{
		"webhook_url:",
		"uses_global_smtp:",
		"writeOnly: true",
		"Write-only secret. Only http/https absolute URLs are accepted.",
	} {
		if !strings.Contains(string(observabilityOpenAPIBody), want) {
			t.Fatalf("observability-api.yaml is missing webhook write-only marker %q", want)
		}
	}
}

func requireControlNotificationComponentSection(t *testing.T, raw, component string) string {
	t.Helper()
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	marker := "    " + component + "\n"
	start := strings.Index(raw, marker)
	if start < 0 {
		t.Fatalf("OpenAPI component %s is missing", component)
	}
	rest := raw[start+len(marker):]
	next := regexp.MustCompile(`(?m)^    [^ \t\r\n]+:\r?$`).FindStringIndex(rest)
	if next == nil {
		return rest
	}
	return rest[:next[0]]
}

func TestControlNotificationComponentSectionIncludesNestedProperties(t *testing.T) {
	const fixture = "components:\n  schemas:\n    NotificationChannel:\n      type: object\n      properties:\n        smtp_password_configured:\n          type: boolean\n    FollowingComponent:\n      type: string\n"
	for _, raw := range []string{fixture, strings.ReplaceAll(fixture, "\n", "\r\n")} {
		section := requireControlNotificationComponentSection(t, raw, "NotificationChannel:")
		if !strings.Contains(section, "smtp_password_configured:") || !strings.Contains(section, "type: boolean") {
			t.Fatal("component scanner cannot see a reintroduced nested legacy property")
		}
		if strings.Contains(section, "FollowingComponent") || strings.Contains(section, "type: string") {
			t.Fatal("component scanner crossed the next component boundary")
		}
	}
}

func TestControlNotificationChannelWriteContractsExcludeLegacySMTP(t *testing.T) {
	type property struct {
		Type        string `json:"type"`
		MinItems    int    `json:"minItems"`
		WriteOnly   bool   `json:"writeOnly"`
		Description string `json:"description"`
	}
	read := func(name string) ([]byte, map[string]property) {
		t.Helper()
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", name))
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Properties map[string]property `json:"properties"`
		}
		if err := json.Unmarshal(body, &schema); err != nil {
			t.Fatal(err)
		}
		return body, schema.Properties
	}

	updateBody, updateProperties := read("control-notification-channel-update.schema.json")
	for _, forbidden := range []string{"uses_global_smtp", "smtp_host", "smtp_port", "smtp_tls", "smtp_from", "smtp_username", "smtp_password"} {
		if _, exists := updateProperties[forbidden]; exists {
			t.Fatalf("Control notification update schema exposes forbidden field %q", forbidden)
		}
	}
	if updateProperties["email_recipients"].MinItems != 1 || !strings.Contains(string(updateBody), "Omission preserves existing recipients") {
		t.Fatal("Control notification update must reject explicit empty recipients while preserving omission")
	}
	migration, exists := updateProperties["migrate_to_global_smtp"]
	if !exists || migration.Type != "boolean" || !migration.WriteOnly || !strings.Contains(migration.Description, "Omission or false preserves") {
		t.Fatalf("Control notification update must expose only the explicit write-only global SMTP migration flag: %#v", migration)
	}

	createBody, _ := read("control-notification-channel-create.schema.json")
	var create struct {
		AllOf []struct {
			Ref string `json:"$ref"`
			Not struct {
				Required []string `json:"required"`
			} `json:"not"`
			Then struct {
				Required []string `json:"required"`
			} `json:"then"`
		} `json:"allOf"`
	}
	if err := json.Unmarshal(createBody, &create); err != nil {
		t.Fatal(err)
	}
	if len(create.AllOf) != 3 || create.AllOf[0].Ref != "control-notification-channel-update.schema.json" || !stringSliceContainsForSchemaTest(create.AllOf[1].Not.Required, "migrate_to_global_smtp") || !stringSliceContainsForSchemaTest(create.AllOf[2].Then.Required, "email_recipients") {
		t.Fatalf("Control email create must require recipients on top of its update shape: %#v", create.AllOf)
	}

	controlOpenAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(controlOpenAPI)
	for _, want := range []string{
		"#/components/schemas/ControlNotificationChannelCreateRequest",
		"#/components/schemas/ControlNotificationChannelUpdateRequest",
		"migrate_to_global_smtp is the only supported delivery-mode transition",
		"an explicitly empty array is invalid",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing Control notification write marker %q", want)
		}
	}
	for _, forbidden := range []string{"    NotificationChannelWriteRequest:", "    NotificationChannelCreateRequest:"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("control-api.yaml still exposes legacy request field/component %q", forbidden)
		}
	}
	start := strings.Index(raw, "    ControlNotificationChannelUpdateRequest:")
	if start < 0 {
		t.Fatal("could not isolate Control notification request components")
	}
	endOffset := strings.Index(raw[start:], "\n    NotificationDeliveryResult:")
	if endOffset < 0 {
		t.Fatal("could not isolate Control notification request components")
	}
	requestComponents := raw[start : start+endOffset]
	if !strings.Contains(requestComponents, "        migrate_to_global_smtp:\n          type: boolean\n          writeOnly: true") {
		t.Fatal("Control notification update OpenAPI component must expose migrate_to_global_smtp as a write-only boolean")
	}
	for _, forbidden := range []string{"        uses_global_smtp:", "        smtp_host:", "        smtp_port:", "        smtp_tls:", "        smtp_from:", "        smtp_username:", "        smtp_password:"} {
		if strings.Contains(requestComponents, forbidden) {
			t.Fatalf("Control notification request components expose forbidden field %q", forbidden)
		}
	}
}

func TestServiceNotificationEmailRelayContracts(t *testing.T) {
	requestBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-notification-email-request.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	responseBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-notification-email-response.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	serviceTokenBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-token-create-request.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serviceTokenBody), `"notifications.email.send"`) {
		t.Fatal("service token create schema is missing the dedicated notification email scope")
	}

	type property struct {
		Type        string `json:"type"`
		MinItems    int    `json:"minItems"`
		MaxItems    int    `json:"maxItems"`
		MinLength   int    `json:"minLength"`
		MaxLength   int    `json:"maxLength"`
		Pattern     string `json:"pattern"`
		UniqueItems bool   `json:"uniqueItems"`
		Const       any    `json:"const"`
	}
	var request struct {
		AdditionalProperties bool                `json:"additionalProperties"`
		Required             []string            `json:"required"`
		Properties           map[string]property `json:"properties"`
	}
	if err := json.Unmarshal(requestBody, &request); err != nil {
		t.Fatal(err)
	}
	if request.AdditionalProperties {
		t.Fatal("service notification email request must reject unknown fields")
	}
	for _, field := range []string{"recipients", "subject", "text"} {
		if !stringSliceContainsForSchemaTest(request.Required, field) {
			t.Fatalf("service notification email request must require %q", field)
		}
	}
	if stringSliceContainsForSchemaTest(request.Required, "html") {
		t.Fatal("service notification email HTML alternative must remain optional")
	}
	recipients := request.Properties["recipients"]
	if recipients.MinItems != 1 || recipients.MaxItems != 20 || !recipients.UniqueItems {
		t.Fatalf("recipient bounds must be 1..20 unique, got %#v", recipients)
	}
	subject := request.Properties["subject"]
	if subject.MinLength != 1 || subject.MaxLength != 200 || !strings.Contains(subject.Pattern, `\r`) || !strings.Contains(subject.Pattern, `\n`) {
		t.Fatalf("subject must be 1..200 code points with CR/LF excluded, got %#v", subject)
	}
	text := request.Properties["text"]
	if text.MinLength != 1 || text.MaxLength != 16384 || !strings.Contains(text.Pattern, `\u0000`) {
		t.Fatalf("text must be 1..16384 with NUL excluded, got %#v", text)
	}
	html := request.Properties["html"]
	if html.MaxLength != 65536 || !strings.Contains(html.Pattern, `\u0000`) {
		t.Fatalf("optional HTML must be at most 65536 with NUL excluded, got %#v", html)
	}

	var response struct {
		Required   []string            `json:"required"`
		Properties map[string]property `json:"properties"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		t.Fatal(err)
	}
	if response.Properties["status"].Const != "sent" || !stringSliceContainsForSchemaTest(response.Required, "recipient_count") {
		t.Fatalf("service notification email response must expose only sent status and recipient count: %#v", response)
	}

	writeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-channel-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var write struct {
		Properties map[string]property `json:"properties"`
	}
	if err := json.Unmarshal(writeBody, &write); err != nil {
		t.Fatal(err)
	}
	if write.Properties["uses_global_smtp"].Const != true {
		t.Fatal("notification write must expose only the global SMTP authority")
	}
	for _, field := range []string{"smtp_host", "smtp_port", "smtp_tls", "smtp_from", "smtp_username", "smtp_password"} {
		if _, exists := write.Properties[field]; exists {
			t.Fatalf("notification write retained removed direct SMTP field %q", field)
		}
	}

	createBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-channel-create.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var create struct {
		AllOf []struct {
			Ref  string `json:"$ref"`
			Then struct {
				Required []string `json:"required"`
			} `json:"then"`
		} `json:"allOf"`
	}
	if err := json.Unmarshal(createBody, &create); err != nil {
		t.Fatal(err)
	}
	if len(create.AllOf) != 2 || create.AllOf[0].Ref != "notification-channel-write.schema.json" || !stringSliceContainsForSchemaTest(create.AllOf[1].Then.Required, "email_recipients") || !stringSliceContainsForSchemaTest(create.AllOf[1].Then.Required, "uses_global_smtp") {
		t.Fatalf("email create schema must require recipients while reusing the update-compatible write schema: %#v", create.AllOf)
	}

	controlOpenAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/services/notifications/email:",
		"requires the dedicated notifications.email.send scope",
		"#/components/schemas/ServiceNotificationEmailRequest",
		"#/components/schemas/ServiceNotificationEmailResponse",
		"#/components/schemas/ControlNotificationChannelCreateRequest",
		"#/components/schemas/NotificationDeliveryResult",
		"status:",
		"const: sent",
		"recipient_count:",
		"smtp_not_configured",
		"smtp_requires_tls",
		"invalid_email_notification",
		"smtp_auth_failed",
		"smtp_recipient_rejected",
		"send_failed",
		"service_type_not_allowed",
		"service_token_not_registered",
		"rate_limited",
		"list_services_failed",
		"app_settings_failed",
		"secret_encryption_key_required",
	} {
		if !strings.Contains(string(controlOpenAPI), want) {
			t.Fatalf("control-api.yaml is missing service email relay marker %q", want)
		}
	}
}

func TestAdministrativeNotificationEventContracts(t *testing.T) {
	if NotificationAdminAudit != NotificationEventType("admin.audit") {
		t.Fatalf("unexpected admin audit event constant %q", NotificationAdminAudit)
	}

	eventBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-event-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	resultBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-delivery-result.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	type eventProperty struct {
		Const     string   `json:"const"`
		Enum      []string `json:"enum"`
		MinLength int      `json:"minLength"`
		MaxLength int      `json:"maxLength"`
		Pattern   string   `json:"pattern"`
		Format    string   `json:"format"`
	}
	var event struct {
		AdditionalProperties bool                     `json:"additionalProperties"`
		Required             []string                 `json:"required"`
		Properties           map[string]eventProperty `json:"properties"`
	}
	if err := json.Unmarshal(eventBody, &event); err != nil {
		t.Fatal(err)
	}
	if event.AdditionalProperties {
		t.Fatal("notification event request must reject metadata and other unknown fields")
	}
	for _, field := range []string{"event_type", "action"} {
		if !stringSliceContainsForSchemaTest(event.Required, field) {
			t.Fatalf("notification event request must require %q", field)
		}
	}
	if _, exists := event.Properties["metadata"]; exists {
		t.Fatal("notification event request must not define metadata")
	}
	for _, field := range []string{"event_type", "severity", "status", "action", "resource_type", "resource_id", "actor_username", "summary", "timestamp"} {
		if _, exists := event.Properties[field]; !exists {
			t.Fatalf("notification event request is missing %q", field)
		}
	}
	if event.Properties["event_type"].Const != "admin.audit" {
		t.Fatalf("notification event type must be admin.audit: %#v", event.Properties["event_type"])
	}
	for field, maxLength := range map[string]int{"status": 64, "action": 128, "resource_type": 80, "resource_id": 160, "actor_username": 80, "summary": 240} {
		property := event.Properties[field]
		if property.MaxLength != maxLength || property.Pattern == "" {
			t.Fatalf("notification event %s must document its safe length and pattern: %#v", field, property)
		}
	}
	const actionPattern = `^[A-Za-z0-9_]+(?:\.[A-Za-z0-9_]+)*$`
	if event.Properties["action"].Pattern != actionPattern {
		t.Fatalf("notification event action pattern = %q, want exact implementation pattern %q", event.Properties["action"].Pattern, actionPattern)
	}
	if event.Properties["timestamp"].Format != "date-time" {
		t.Fatalf("notification event timestamp must use date-time format: %#v", event.Properties["timestamp"])
	}
	for _, marker := range []string{"Raw tokens", "credentials", "webhook URLs", "passwords", "authorization values"} {
		if !strings.Contains(string(eventBody), marker) {
			t.Fatalf("notification event schema is missing secret-boundary marker %q", marker)
		}
	}

	var result struct {
		AdditionalProperties bool                     `json:"additionalProperties"`
		Required             []string                 `json:"required"`
		Properties           map[string]eventProperty `json:"properties"`
	}
	if err := json.Unmarshal(resultBody, &result); err != nil {
		t.Fatal(err)
	}
	if result.AdditionalProperties {
		t.Fatal("notification delivery result must reject undeclared response fields")
	}
	for _, field := range []string{"event_type", "channel", "target", "status"} {
		if !stringSliceContainsForSchemaTest(result.Required, field) {
			t.Fatalf("notification delivery result must require %q", field)
		}
	}
	if !stringSliceContainsForSchemaTest(result.Properties["event_type"].Enum, "admin.audit") {
		t.Fatalf("notification delivery result event enum is missing admin.audit: %#v", result.Properties["event_type"])
	}

	observabilityOpenAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "observability-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/notification-events:",
		"operationId: createNotificationEvent",
		"#/components/schemas/NotificationEventWriteRequest",
		"#/components/schemas/NotificationDeliveryResult",
		"required: [event_type, action]",
		"const: admin.audit",
		"metadata and all other unknown properties are rejected",
		"event_id and outbox/idempotency semantics are not part of this endpoint",
		"invalid_notification_event",
		"missing_admin_scope",
		"rate_limited",
		"rate_limit_unavailable",
	} {
		if !strings.Contains(string(observabilityOpenAPI), want) {
			t.Fatalf("observability-api.yaml is missing administrative notification marker %q", want)
		}
	}
}
