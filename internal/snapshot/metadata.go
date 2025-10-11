package snapshot

import "time"

type Metadata struct {
	Id         string           `json:"id"`
	CreatedAt  time.Time        `json:"created_at"`
	ModifiedAt time.Time        `json:"modified_at,omitempty"`
	Provider   *ProviderInfo    `json:"provider"`
	Context    *SnapshotContext `json:"context,omitempty"`
}

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

type SnapshotContext struct {
	Description string `json:"description,omitempty"`
}
