package contracts

import (
	"time"
)

// ServiceNotificationEmailRequest asks Control Panel to deliver one text message
// and, optionally, an HTML alternative with its globally managed SMTP settings.
// This service-authenticated
// request is restricted to a registered Observability service token that has
// the dedicated notifications.email.send scope.
type ServiceNotificationEmailRequest struct {
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
	Text       string   `json:"text"`
	HTML       string   `json:"html,omitempty"`
}

// ServiceNotificationEmailResponse intentionally reports only a count so raw
// recipient addresses and SMTP settings never cross the response boundary.
type ServiceNotificationEmailResponse struct {
	Status         string `json:"status"`
	RecipientCount int    `json:"recipient_count"`
}

type YouTubeLiveNotificationRequest struct {
	EventID  string `json:"event_id"`
	WatchURL string `json:"watch_url"`
}

type YouTubeLiveNotificationResponse struct {
	Status      string `json:"status"`
	MessageID   string `json:"message_id"`
	AlreadySent bool   `json:"already_sent"`
}

type ServiceNotificationError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type NotificationEventType string

const (
	NotificationStreamStarted              NotificationEventType = "stream.started"
	NotificationStreamLive                 NotificationEventType = "stream.live"
	NotificationStreamCompleted            NotificationEventType = "stream.completed"
	NotificationStreamFailed               NotificationEventType = "stream.failed"
	NotificationStreamWarning              NotificationEventType = "stream.warning"
	NotificationStreamError                NotificationEventType = "stream.error"
	NotificationIncidentOpened             NotificationEventType = "incident.opened"
	NotificationIncidentUpdated            NotificationEventType = "incident.updated"
	NotificationIncidentResolved           NotificationEventType = "incident.resolved"
	NotificationDiagnosticCreated          NotificationEventType = "diagnostic.created"
	NotificationRemediationPendingApproval NotificationEventType = "remediation.pending_approval"
	NotificationRemediationExecuted        NotificationEventType = "remediation.executed"
	NotificationArchiveUploadCompleted     NotificationEventType = "archive.upload.completed"
	NotificationArchiveUploadFailed        NotificationEventType = "archive.upload.failed"
	NotificationServiceOffline             NotificationEventType = "service.offline"
	NotificationServiceRecovered           NotificationEventType = "service.recovered"
	NotificationAdminAudit                 NotificationEventType = "admin.audit"
)

// NotificationEventWriteRequest is the secret-free event envelope accepted by
// Observability's notification event endpoint. Metadata is deliberately not
// part of this contract; service_id/details carry only display-safe context.
type NotificationEventWriteRequest struct {
	EventType     NotificationEventType `json:"event_type"`
	Severity      string                `json:"severity,omitempty"`
	Status        string                `json:"status,omitempty"`
	Action        string                `json:"action"`
	ServiceID     string                `json:"service_id,omitempty"`
	ResourceType  string                `json:"resource_type,omitempty"`
	ResourceID    string                `json:"resource_id,omitempty"`
	ActorUsername string                `json:"actor_username,omitempty"`
	Summary       string                `json:"summary,omitempty"`
	Details       string                `json:"details,omitempty"`
	Timestamp     string                `json:"timestamp,omitempty"`
}

// NotificationDeliveryResult is a secret-safe delivery attempt returned by
// notification-events. Target and Error must already be masked or sanitized.
type NotificationDeliveryResult struct {
	EventType NotificationEventType `json:"event_type"`
	Channel   string                `json:"channel"`
	Target    string                `json:"target"`
	Status    string                `json:"status"`
	Error     string                `json:"error,omitempty"`
}

type NotificationChannel struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	Enabled           bool      `json:"enabled"`
	UsesGlobalSMTP    bool      `json:"uses_global_smtp"`
	MaskedWebhookURL  string    `json:"masked_webhook_url,omitempty"`
	MaskedEmailTarget string    `json:"masked_email_target,omitempty"`
	SeverityFilter    []string  `json:"severity_filter,omitempty"`
	EventTypeFilter   []string  `json:"event_type_filter,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type NotificationChannelWriteRequest struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Enabled         bool     `json:"enabled"`
	WebhookURL      string   `json:"webhook_url,omitempty"`
	EmailRecipients []string `json:"email_recipients,omitempty"`
	UsesGlobalSMTP  *bool    `json:"uses_global_smtp,omitempty"`
	SeverityFilter  []string `json:"severity_filter,omitempty"`
	EventTypeFilter []string `json:"event_type_filter,omitempty"`
}

// ControlNotificationChannelUpdateRequest is the browser-facing Control Panel
// write shape. It deliberately excludes per-channel SMTP configuration and the
// global/legacy SMTP mode selector. Omitted recipients preserve the existing
// masked recipient set and delivery mode.
type ControlNotificationChannelUpdateRequest struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Enabled         bool     `json:"enabled"`
	WebhookURL      string   `json:"webhook_url,omitempty"`
	EmailRecipients []string `json:"email_recipients,omitempty"`
	SeverityFilter  []string `json:"severity_filter,omitempty"`
	EventTypeFilter []string `json:"event_type_filter,omitempty"`
}

// ControlNotificationChannelCreateRequest has the same wire fields as an
// update. The create schema additionally requires recipients for email and the
// backend always selects globally managed SMTP for new email channels.
type ControlNotificationChannelCreateRequest = ControlNotificationChannelUpdateRequest

type NotificationDelivery struct {
	ID         string                `json:"id"`
	EventType  NotificationEventType `json:"event_type"`
	Channel    string                `json:"channel"`
	Target     string                `json:"target"`
	IncidentID string                `json:"incident_id,omitempty"`
	Status     string                `json:"status"`
	Error      string                `json:"error,omitempty"`
	Metadata   map[string]any        `json:"metadata,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`
}
