package contracts

import (
	"encoding/json"
	"testing"
)

func TestSystemSettingsWriteBaselineAndUnconfiguredAuthority(t *testing.T) {
	dependencies := []string{"system-settings-revision.schema.json", "resource-updated-at.schema.json", "security-settings.schema.json"}
	for _, test := range []struct {
		name string
		body map[string]any
	}{
		{"app-settings-write.schema.json", map[string]any{"app_name": "Panel", "timezone": "Asia/Tokyo"}},
		{"security-settings-write.schema.json", map[string]any{"password_min_length": float64(12), "password_hash": "argon2id", "login_lockout_threshold": float64(5), "session_idle_timeout_min": float64(30), "session_absolute_lifetime_h": float64(12), "remember_me_enabled": false, "mfa_mode": "disabled"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema := compileContractJSONSchema(t, test.name, dependencies...)
			if err := schema.Validate(test.body); err != nil {
				t.Fatalf("legacy omission rejected: %v", err)
			}
			for _, revision := range []string{"unconfigured", "2030-01-01T01:02:03Z", "2030-01-01T01:02:03.123456Z"} {
				test.body["expected_updated_at"] = revision
				if err := schema.Validate(test.body); err != nil {
					t.Fatalf("valid system baseline rejected: %v", err)
				}
			}
			for _, invalid := range []any{"", "2030-01-01T01:02:03+00:00", "2030-01-01T01:02:03.100000Z", "2030-01-01T01:02:03.1234567Z", nil} {
				test.body["expected_updated_at"] = invalid
				if schema.Validate(test.body) == nil {
					t.Fatalf("invalid system baseline accepted: %v", invalid)
				}
			}
		})
	}
	stamp := "unconfigured"
	for _, request := range []any{
		AppSettingsWriteRequest{AppName: "Panel", Timezone: "Asia/Tokyo", ExpectedUpdatedAt: &stamp},
		SecuritySettingsWriteRequest{SecuritySettings: SecuritySettings{PasswordMinLength: 12, PasswordHash: "argon2id", LoginLockoutThreshold: 5, SessionIdleTimeoutMin: 30, SessionAbsoluteLifetimeH: 12, MFAMode: "disabled"}, ExpectedUpdatedAt: &stamp},
	} {
		body, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err = json.Unmarshal(body, &decoded); err != nil || decoded["expected_updated_at"] != stamp {
			t.Fatal("system write DTO lost the original initial baseline")
		}
	}
}

func TestSystemSettingsReadSchemasAcceptStableInitialRevision(t *testing.T) {
	for _, test := range []struct {
		name string
		body map[string]any
	}{
		{"app-settings-public.schema.json", map[string]any{"app_name": "Panel", "timezone": "Asia/Tokyo"}},
		{"app-settings-manage.schema.json", map[string]any{"app_name": "Panel", "timezone": "Asia/Tokyo", "smtp_enabled": false, "smtp_starttls": true}},
		{"security-settings.schema.json", map[string]any{"password_min_length": float64(12), "password_hash": "argon2id", "login_lockout_threshold": float64(5), "session_idle_timeout_min": float64(30), "session_absolute_lifetime_h": float64(12), "remember_me_enabled": false, "mfa_mode": "disabled"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema := compileContractJSONSchema(t, test.name, "system-settings-revision.schema.json", "resource-updated-at.schema.json")
			test.body["updated_at"] = "unconfigured"
			if err := schema.Validate(test.body); err != nil {
				t.Fatal(err)
			}
			test.body["updated_at"] = "2030-01-01T01:02:03.123456Z"
			if err := schema.Validate(test.body); err != nil {
				t.Fatal(err)
			}
		})
	}
}
