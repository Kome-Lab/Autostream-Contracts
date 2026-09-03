package contracts

import (
	"encoding/hex"
	"encoding/json"
	"errors"
)

// MarshalJSON preserves host/image component fields while keeping the
// independent Updater's Docker-bundle entry metadata-only. Updater metadata is
// not permission to download an image or execute/publish a host release.
func (component ReleaseManifestComponent) MarshalJSON() ([]byte, error) {
	if component.Service != "updater" {
		type componentWire ReleaseManifestComponent
		return json.Marshal(componentWire(component))
	}
	commit, err := hex.DecodeString(component.Commit)
	if err != nil || len(commit) != 20 || component.ProtocolMajor != 2 ||
		component.SourceVersion != "" || component.Image != "" || component.ManifestDigest != "" ||
		component.Artifacts != nil || component.PlatformDigests != nil ||
		component.RollbackCompatible || component.DatabaseSchema != "" {
		return nil, errors.New("invalid updater release component")
	}
	return json.Marshal(struct {
		Service       string `json:"service"`
		Commit        string `json:"commit"`
		ProtocolMajor int    `json:"protocol_major"`
	}{Service: component.Service, Commit: component.Commit, ProtocolMajor: component.ProtocolMajor})
}
