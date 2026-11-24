package mediarelease

import (
	"reflect"
	"slices"
)

// VersionTuple is negotiated as one unit; a participant may advertise several
// tuples, but individually acceptable fields from different tuples never join.
type VersionTuple struct {
	Family          string `json:"family"`
	WireVersion     string `json:"wire_version"`
	ProfileID       string `json:"profile_id"`
	ProfileRevision string `json:"profile_revision"`
	EncodingID      string `json:"encoding_id"`
}

// Participant identifies a role, its real incarnation and the protected operation.
type Participant struct {
	Role     string   `json:"role"`
	Owner    Owner    `json:"owner"`
	Original Original `json:"original"`
	Scope    Scope    `json:"scope"`
}

// Each authenticated snapshot asserts every listed operation for every listed
// tuple. A participant with version-specific operations advertises the
// intersection or a single tuple. The negotiator never combines two offers.
// Authenticated is set only by the caller's trusted identity adapter.
type Capability struct {
	Participant   Participant    `json:"participant"`
	Versions      []VersionTuple `json:"versions"`
	Operations    []string       `json:"operations"`
	SnapshotRef   Ref            `json:"snapshot_ref"`
	ObservedAt    string         `json:"observed_at"`
	ExpiresAt     string         `json:"expires_at"`
	Authenticated bool           `json:"-"`
}

// RequiredOperations returns a new list owned by the caller.
func RequiredOperations(role string) []string {
	switch role {
	case "CP":
		return []string{"getMediaReleaseCapabilities", "issueStartAdmissionProof", "issueMediaReleasePermit", "revokeMediaReleasePermit", "inspectMediaReleaseRecord"}
	case "WORKER":
		return []string{"getMediaReleaseCapabilities", "bindNonMediaWorker", "inspectMediaReleaseRecord"}
	case "ENCODER":
		return []string{"getMediaReleaseCapabilities", "attestAppliedCover", "inspectMediaReleaseRecord"}
	case "GATE":
		return []string{"getMediaReleaseCapabilities", "openMediaRelease", "fenceMediaRelease", "inspectMediaReleaseRecord", "inspectMediaReleaseReceipt"}
	}
	return nil
}

// Negotiate compares an authoritative required roster with authenticated offers
// for this exact operation. Coherent selects the version only; it never grants
// production or transport authority. Old or missing owners leave the lane held.
func Negotiate(required []Participant, offered []Capability, now string) Decision {
	if len(required) != len(offered) || len(required) < 4 || len(required) > 64 || !validTimestamp(now) {
		return Held
	}
	want := VersionTuple{Family, WireVersion, ProfileID, ProfileRevision, EncodingID}
	roles := make(map[string]bool)
	type roleOwner struct {
		role  string
		owner Owner
	}
	matched := make(map[roleOwner]bool)
	type snapshotVersion struct {
		key      Key
		revision string
	}
	seenSnapshots := make(map[snapshotVersion]bool)
	participants := make(map[Peer]string)
	var gateMatches, workerMatches, encoderMatches bool
	for _, need := range required {
		ops := RequiredOperations(need.Role)
		id := roleOwner{need.Role, need.Owner}
		if len(ops) == 0 || matched[id] || need.Role != "GATE" && roles[need.Role] ||
			validateValue(need.Owner) != nil || validateValue(need.Original) != nil || validateValue(need.Scope) != nil ||
			need.Original != required[0].Original || !reflect.DeepEqual(need.Scope, required[0].Scope) {
			return Held
		}
		var offer *Capability
		for i := range offered {
			if reflect.DeepEqual(offered[i].Participant, need) {
				if offer != nil {
					return Conflict
				}
				offer = &offered[i]
			}
		}
		if offer == nil || !offer.Authenticated || !validTimestamp(offer.ObservedAt) || !validTimestamp(offer.ExpiresAt) ||
			offer.ObservedAt > now || offer.ExpiresAt <= now || validateValue(offer.SnapshotRef) != nil ||
			offer.SnapshotRef.SourceOwner != need.Owner.Peer() ||
			offer.SnapshotRef.Representation != "OWNER_SEALED_BYTES" || offer.SnapshotRef.SourceType != "CAPABILITY_SNAPSHOT" ||
			offer.SnapshotRef.SourceKey.Kind != "CAPABILITY_SNAPSHOT" ||
			len(offer.Versions) > 64 || len(offer.Operations) > 64 || !slices.Contains(offer.Versions, want) {
			return Held
		}
		// The wire has one participant/role per offer. The same immutable
		// snapshot cannot be authoritative for two different role bodies.
		version := snapshotVersion{offer.SnapshotRef.SourceKey, offer.SnapshotRef.SourceRevision}
		if seenSnapshots[version] {
			return Conflict
		}
		seenSnapshots[version] = true
		if need.Role != "GATE" {
			if _, used := participants[need.Owner.Peer()]; used {
				return Conflict
			}
			participants[need.Owner.Peer()] = need.Role
		}
		seenVersions := make(map[VersionTuple]bool)
		for _, v := range offer.Versions {
			if seenVersions[v] || !validID(v.Family) || !validID(v.EncodingID) || !validID(v.ProfileID) ||
				!validUint(v.WireVersion, true) || !validUint(v.ProfileRevision, true) {
				return Held
			}
			seenVersions[v] = true
		}
		advertised := make(map[string]bool)
		for _, op := range offer.Operations {
			if !validID(op) || advertised[op] {
				return Held
			}
			advertised[op] = true
		}
		for _, op := range ops {
			if !advertised[op] {
				return Held
			}
		}
		roles[need.Role] = true
		matched[id] = true
		if need.Owner == need.Scope.TargetOwner {
			switch need.Role {
			case "GATE":
				gateMatches = true
			case "WORKER":
				workerMatches = true
			case "ENCODER":
				encoderMatches = true
			}
		}
	}
	if !roles["CP"] || !roles["WORKER"] || !roles["ENCODER"] || !gateMatches {
		return Held
	}
	switch required[0].Scope.Route {
	case "WORKER_SRT", "WORKER_DEEPGRAM":
		if !workerMatches {
			return Held
		}
	case "ENCODER_DIRECT":
		if !encoderMatches {
			return Held
		}
	}
	return Coherent
}
