package contracts

// NodeMetadataWriteRequest is a Panel metadata edit. Monitoring timestamps and
// heartbeat observations are not an editing authority.
type NodeMetadataWriteRequest struct {
	ExpectedMetadataRevision *string `json:"expected_metadata_revision,omitempty"`
	Name                     *string `json:"name,omitempty"`
	ServiceName              *string `json:"service_name,omitempty"`
	Description              *string `json:"description,omitempty"`
	Host                     *string `json:"host,omitempty"`
	Port                     *int    `json:"port,omitempty"`
	SSLEnabled               *bool   `json:"ssl_enabled,omitempty"`
	PublicURL                *string `json:"public_url,omitempty"`
}
