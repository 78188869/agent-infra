package model

// Credential type constants.
const (
	CredentialTypeGitToken    = "git_token"
	CredentialTypeDevOpsToken = "devops_token"
)

// Credential stores encrypted user credentials for third-party services.
type Credential struct {
	BaseModel
	UserID    string `gorm:"type:char(36);not null;uniqueIndex:idx_user_type" json:"user_id"`
	Type      string `gorm:"type:varchar(32);not null;uniqueIndex:idx_user_type" json:"type"`
	Encrypted string `gorm:"type:text;not null" json:"-"`
}

// ValidCredentialTypes returns the list of valid credential types.
func ValidCredentialTypes() []string {
	return []string{CredentialTypeGitToken, CredentialTypeDevOpsToken}
}

// IsValidCredentialType checks if the given type is valid.
func IsValidCredentialType(t string) bool {
	for _, vt := range ValidCredentialTypes() {
		if vt == t {
			return true
		}
	}
	return false
}
