package contracts

import (
	"time"
)

type PasskeyCredential struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	Name             string   `json:"name"`
	CredentialIDHash string   `json:"credential_id_hash"`
	SignCount        uint32   `json:"sign_count"`
	Transports       []string `json:"transports,omitempty"`
	AAGUID           string   `json:"aaguid,omitempty"`
	BackupEligible   bool     `json:"backup_eligible,omitempty"`
	BackedUp         bool     `json:"backed_up,omitempty"`
	CreatedAt        string   `json:"created_at,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	LastUsedAt       string   `json:"last_used_at,omitempty"`
}

type PasskeyRegistrationStartRequest struct {
	DisplayName string `json:"display_name,omitempty"`
}

type PasskeyRegistrationStartResponse struct {
	RegistrationToken string                 `json:"registration_token"`
	ExpiresAt         string                 `json:"expires_at"`
	PublicKey         PasskeyCreationOptions `json:"public_key"`
}

type PasskeyRegistrationFinishRequest struct {
	RegistrationToken string         `json:"registration_token"`
	Name              string         `json:"name,omitempty"`
	Credential        map[string]any `json:"credential"`
}

type PasskeyLoginStartRequest struct {
	Username string `json:"username,omitempty"`
}

type PasskeyLoginStartResponse struct {
	ChallengeToken string         `json:"challenge_token"`
	ExpiresAt      string         `json:"expires_at"`
	PublicKey      map[string]any `json:"public_key"`
}

type PasskeyLoginFinishRequest struct {
	ChallengeToken string         `json:"challenge_token"`
	Credential     map[string]any `json:"credential"`
}

type PasskeyCreationOptions struct {
	Challenge              string                        `json:"challenge"`
	RP                     PasskeyRelyingParty           `json:"rp"`
	User                   PasskeyUser                   `json:"user"`
	PubKeyCredParams       []PasskeyCredentialParameter  `json:"pubKeyCredParams"`
	Timeout                int                           `json:"timeout"`
	Attestation            string                        `json:"attestation"`
	AuthenticatorSelection PasskeyAuthenticatorSelection `json:"authenticatorSelection"`
}

type PasskeyRelyingParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PasskeyUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type PasskeyCredentialParameter struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

type PasskeyAuthenticatorSelection struct {
	ResidentKey      string `json:"residentKey"`
	UserVerification string `json:"userVerification"`
}

type SessionRefreshResponse struct {
	Status            string    `json:"status"`
	IdleExpiresAt     time.Time `json:"idle_expires_at"`
	AbsoluteExpiresAt time.Time `json:"absolute_expires_at"`
}

type PublicAppSettings struct {
	AppName                      string `json:"app_name"`
	Timezone                     string `json:"timezone"`
	TurnstileEnabled             bool   `json:"turnstile_enabled,omitempty"`
	TurnstileSiteKey             string `json:"turnstile_site_key,omitempty"`
	TurnstileConfigured          bool   `json:"turnstile_configured,omitempty"`
	GoogleAnalyticsEnabled       bool   `json:"google_analytics_enabled,omitempty"`
	GoogleAnalyticsMeasurementID string `json:"google_analytics_measurement_id,omitempty"`
	UpdatedAt                    string `json:"updated_at,omitempty"`
}

type ManagedAppSettings struct {
	AppName                      string `json:"app_name"`
	Timezone                     string `json:"timezone"`
	GoogleAnalyticsEnabled       bool   `json:"google_analytics_enabled,omitempty"`
	GoogleAnalyticsMeasurementID string `json:"google_analytics_measurement_id,omitempty"`
	SMTPEnabled                  bool   `json:"smtp_enabled"`
	SMTPHost                     string `json:"smtp_host,omitempty"`
	SMTPPort                     int    `json:"smtp_port,omitempty"`
	SMTPStartTLS                 bool   `json:"smtp_starttls"`
	SMTPFrom                     string `json:"smtp_from,omitempty"`
	SMTPUsername                 string `json:"smtp_username,omitempty"`
	SMTPPasswordConfigured       bool   `json:"smtp_password_configured,omitempty"`
	TurnstileEnabled             bool   `json:"turnstile_enabled,omitempty"`
	TurnstileSiteKey             string `json:"turnstile_site_key,omitempty"`
	TurnstileConfigured          bool   `json:"turnstile_configured,omitempty"`
	UpdatedAt                    string `json:"updated_at,omitempty"`
}

type AppSettingsWriteRequest struct {
	AppName                      string `json:"app_name"`
	Timezone                     string `json:"timezone"`
	GoogleAnalyticsEnabled       bool   `json:"google_analytics_enabled,omitempty"`
	GoogleAnalyticsMeasurementID string `json:"google_analytics_measurement_id,omitempty"`
	SMTPEnabled                  bool   `json:"smtp_enabled"`
	SMTPHost                     string `json:"smtp_host,omitempty"`
	SMTPPort                     int    `json:"smtp_port,omitempty"`
	SMTPStartTLS                 bool   `json:"smtp_starttls"`
	SMTPFrom                     string `json:"smtp_from,omitempty"`
	SMTPUsername                 string `json:"smtp_username,omitempty"`
	SMTPPassword                 string `json:"smtp_password,omitempty"`
	TurnstileEnabled             bool   `json:"turnstile_enabled,omitempty"`
	TurnstileSiteKey             string `json:"turnstile_site_key,omitempty"`
	TurnstileSecret              string `json:"turnstile_secret,omitempty"`
}

type SecuritySettings struct {
	PasswordMinLength        int      `json:"password_min_length"`
	PasswordHash             string   `json:"password_hash"`
	LoginLockoutThreshold    int      `json:"login_lockout_threshold"`
	SessionIdleTimeoutMin    int      `json:"session_idle_timeout_min"`
	SessionAbsoluteLifetimeH int      `json:"session_absolute_lifetime_h"`
	RememberMeEnabled        bool     `json:"remember_me_enabled"`
	MFAMode                  string   `json:"mfa_mode"`
	MFARequiredRoles         []string `json:"mfa_required_roles,omitempty"`
	MFASupportedMethods      []string `json:"mfa_supported_methods,omitempty"`
	PasskeyStatus            string   `json:"passkey_status,omitempty"`
	UpdatedAt                string   `json:"updated_at,omitempty"`
}

type SecretStatus struct {
	Name        string `json:"name"`
	Configured  bool   `json:"configured"`
	Fingerprint string `json:"fingerprint,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type SecretUpdateRequest struct {
	Value string `json:"value"`
}

type OAuthProvider struct {
	ID                     string    `json:"id"`
	ProviderType           string    `json:"provider_type"`
	Name                   string    `json:"name"`
	Enabled                bool      `json:"enabled"`
	ClientID               string    `json:"client_id"`
	ClientSecretConfigured bool      `json:"client_secret_configured"`
	Scopes                 []string  `json:"scopes"`
	AllowedDomains         []string  `json:"allowed_domains"`
	AutoProvision          bool      `json:"auto_provision"`
	DefaultRoleIDs         []string  `json:"default_role_ids,omitempty"`
	RedirectURI            string    `json:"redirect_uri"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type OAuthProviderWriteRequest struct {
	ProviderType   string   `json:"provider_type"`
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	AutoProvision  bool     `json:"auto_provision,omitempty"`
	DefaultRoleIDs []string `json:"default_role_ids,omitempty"`
	// RedirectURI must match AUTOSTREAM_PUBLIC_URL scheme/host in production.
	// /auth/oauth/callback is used for login providers.
	// /integrations/oauth-accounts/callback is only for Google Drive/YouTube connected accounts.
	RedirectURI string `json:"redirect_uri"`
}

// OAuthAccountPurpose is derived from the effective OAuth scopes. It is
// returned by connected-account reads and OAuth consent starts; it never
// contains an OAuth scope or credential.
type OAuthAccountPurpose string

const (
	OAuthAccountPurposeDrive        OAuthAccountPurpose = "drive"
	OAuthAccountPurposeYouTube      OAuthAccountPurpose = "youtube"
	OAuthAccountPurposeDriveYouTube OAuthAccountPurpose = "drive_youtube"
	OAuthAccountPurposeUnknown      OAuthAccountPurpose = "unknown"
)

type OAuthAccount struct {
	ID                               string              `json:"id"`
	ProviderID                       string              `json:"provider_id"`
	ProviderType                     string              `json:"provider_type"`
	ProviderName                     string              `json:"provider_name,omitempty"`
	AccountLabel                     string              `json:"account_label"`
	AccountPurpose                   OAuthAccountPurpose `json:"account_purpose"`
	DisplayName                      string              `json:"display_name,omitempty"`
	Subject                          string              `json:"subject,omitempty"`
	Email                            string              `json:"email,omitempty"`
	Scopes                           []string            `json:"scopes"`
	RefreshTokenConfigured           bool                `json:"refresh_token_configured"`
	TokenFingerprint                 string              `json:"token_fingerprint,omitempty"`
	RefreshTokenUpdatedAt            string              `json:"refresh_token_updated_at,omitempty"`
	AccessTokenRefreshedAt           string              `json:"access_token_refreshed_at,omitempty"`
	AccessTokenRefreshAttemptedAt    string              `json:"access_token_refresh_attempted_at,omitempty"`
	AccessTokenRefreshFailedAt       string              `json:"access_token_refresh_failed_at,omitempty"`
	AccessTokenRefreshFailureCode    string              `json:"access_token_refresh_failure_code,omitempty"`
	AccessTokenRefreshRelinkRequired bool                `json:"access_token_refresh_relink_required"`
	CreatedAt                        time.Time           `json:"created_at"`
	UpdatedAt                        time.Time           `json:"updated_at"`
}

type OAuthAccountConnectionStartRequest struct {
	ProviderID     string `json:"provider_id,omitempty"`
	OAuthAccountID string `json:"oauth_account_id,omitempty"`
	AccountLabel   string `json:"account_label,omitempty"`
	AccountPurpose string `json:"account_purpose,omitempty"`
	RedirectAfter  string `json:"redirect_after,omitempty"`
}

type OAuthAccountConnectionStartResponse struct {
	Provider         OAuthLoginProvider  `json:"provider"`
	AuthorizationURL string              `json:"authorization_url"`
	State            string              `json:"state"`
	Nonce            string              `json:"nonce"`
	ExpiresAt        time.Time           `json:"expires_at"`
	AccountLabel     string              `json:"account_label"`
	AccountPurpose   OAuthAccountPurpose `json:"account_purpose"`
	Relink           bool                `json:"relink"`
	Scopes           []string            `json:"scopes"`
}

type OAuthAccountConnectionCallbackRequest struct {
	ProviderID   string `json:"provider_id,omitempty"`
	State        string `json:"state"`
	Code         string `json:"code"`
	AccountLabel string `json:"account_label,omitempty"`
}

type OAuthLoginProvider struct {
	ID           string   `json:"id"`
	ProviderType string   `json:"provider_type"`
	Name         string   `json:"name"`
	Scopes       []string `json:"scopes"`
	RedirectURI  string   `json:"redirect_uri"`
}

type OAuthLoginStartRequest struct {
	RedirectAfter string `json:"redirect_after,omitempty"`
}

type OAuthLoginStartResponse struct {
	Provider         OAuthLoginProvider `json:"provider"`
	AuthorizationURL string             `json:"authorization_url"`
	State            string             `json:"state"`
	Nonce            string             `json:"nonce"`
	ExpiresAt        time.Time          `json:"expires_at"`
}

type OAuthLoginCallbackRequest struct {
	ProviderID string `json:"provider_id,omitempty"`
	State      string `json:"state"`
	Code       string `json:"code"`
}

type OAuthUserLink struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ProviderID   string    `json:"provider_id"`
	ProviderType string    `json:"provider_type"`
	Subject      string    `json:"subject"`
	Email        string    `json:"email,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
