package schema

type DataClass string

const (
	PublicSynthetic DataClass = "public_synthetic"
	PublicProject   DataClass = "public_project"
	PrivateProject  DataClass = "private_project"
	SecretDenied    DataClass = "secret_denied"
)

type ProviderHandshake struct {
	ProtocolVersion string   `json:"protocol_version"`
	Provider        string   `json:"provider"`
	Capabilities    []string `json:"capabilities"`
	MaxInputBytes   int64    `json:"max_input_bytes"`
	SupportsResume  bool     `json:"supports_resume"`
	NetworkAccess   string   `json:"network_access"`
}

func (h ProviderHandshake) Valid() bool {
	if h.ProtocolVersion != "1.0" || h.Provider == "" || h.MaxInputBytes < 1 {
		return false
	}
	switch h.NetworkAccess {
	case "none", "required", "unknown":
		return len(h.Capabilities) > 0
	default:
		return false
	}
}

type ApprovalBinding struct {
	SchemaVersion   string    `json:"schema_version"`
	CandidateDigest string    `json:"candidate_digest"`
	DataClass       DataClass `json:"data_class"`
	Action          string    `json:"action"`
	PolicyDigest    string    `json:"policy_digest"`
	ScanDigest      string    `json:"scan_digest"`
	ApprovedAt      string    `json:"approved_at"`
}
