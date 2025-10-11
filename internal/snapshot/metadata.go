package snapshot

type ProviderInfo struct {
	Name             string   `json:"name"`
	DetectedSource   string   `json:"detected_source"`
	DetectedVersion  string   `json:"detected_version"`
	NormalizedSource string   `json:"normalized_source,omitempty"`
	IsLocalBuild     bool     `json:"is_local_build"`
	SchemaFile       string   `json:"schema_file,omitempty"`
	Binary           *Binary  `json:"binary,omitempty"`
	GitInfo          *GitInfo `json:"git_info,omitempty"`
}

type Metadata struct {
	Id        string       `json:"id"`
	CreatedAt string       `json:"created_at"`
	Provider  ProviderInfo `json:"provider"`
}

type GitInfo struct {
	Commit    string `json:"commit,omitempty"`
	Branch    string `json:"branch,omitempty"`
	IsDirty   bool   `json:"is_dirty,omitempty"`
	Remote    string `json:"remote,omitempty"`
	CommitMsg string `json:"commit_message,omitempty"`
}

type Binary struct {
	OriginalPath       string `json:"original_path"`
	SnapshotBinaryPath string `json:"snapshot_binary_path"`
	Hash               string `json:"hash"`
	Size               int64  `json:"size"`
}
