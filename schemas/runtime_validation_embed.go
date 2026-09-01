// Package schemas exposes the canonical JSON Schemas that runtime contract
// validators must apply before decoding presence-losing Go DTOs.
package schemas

import "embed"

// RuntimeValidationFS embeds the canonical source files directly. Keeping the
// files in this directory avoids a second, generated schema authority.
//
//go:embed discord-bot-start-job-request.schema.json encoder-video-cover-apply-request.schema.json encoder-video-cover-apply-response.schema.json encoder-video-cover-runtime-state.schema.json
var RuntimeValidationFS embed.FS
