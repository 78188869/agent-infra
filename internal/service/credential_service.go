package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/agent-infra/internal/config"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
)

// CredentialInfo represents a credential's metadata (no secret values).
type CredentialInfo struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// StoreCredentialRequest represents the request to store a credential.
type StoreCredentialRequest struct {
	Type  string `json:"type" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// CredentialService defines the interface for credential business operations.
type CredentialService interface {
	Store(ctx context.Context, userID string, req *StoreCredentialRequest) (*CredentialInfo, error)
	Get(ctx context.Context, userID, credType string) (string, error)
	Delete(ctx context.Context, userID, credType string) error
	List(ctx context.Context, userID string) ([]*CredentialInfo, error)
	BuildSandboxEnv(ctx context.Context, userID string) (map[string]string, error)
}

type credentialService struct {
	repo      repository.CredentialRepository
	encryptor config.Encryptor
}

// NewCredentialService creates a new CredentialService instance.
func NewCredentialService(repo repository.CredentialRepository, encryptor config.Encryptor) CredentialService {
	return &credentialService{repo: repo, encryptor: encryptor}
}

func (s *credentialService) Store(ctx context.Context, userID string, req *StoreCredentialRequest) (*CredentialInfo, error) {
	if !model.IsValidCredentialType(req.Type) {
		return nil, errors.NewBadRequestError(fmt.Sprintf("invalid credential type: %s", req.Type))
	}
	if req.Value == "" {
		return nil, errors.NewBadRequestError("credential value is required")
	}

	encrypted, err := s.encryptor.Encrypt(req.Value)
	if err != nil {
		return nil, errors.NewInternalError("failed to encrypt credential: " + err.Error())
	}

	cred := &model.Credential{
		UserID:    userID,
		Type:      req.Type,
		Encrypted: encrypted,
	}

	if err := s.repo.Store(ctx, cred); err != nil {
		return nil, err
	}

	return &CredentialInfo{ID: cred.ID.String(), Type: cred.Type}, nil
}

func (s *credentialService) Get(ctx context.Context, userID, credType string) (string, error) {
	if !model.IsValidCredentialType(credType) {
		return "", errors.NewBadRequestError(fmt.Sprintf("invalid credential type: %s", credType))
	}

	cred, err := s.repo.GetByUserAndType(ctx, userID, credType)
	if err != nil {
		return "", err
	}
	if cred == nil {
		return "", errors.NewNotFoundError("credential not found")
	}

	plaintext, err := s.encryptor.Decrypt(cred.Encrypted)
	if err != nil {
		return "", errors.NewInternalError("failed to decrypt credential: " + err.Error())
	}

	return plaintext, nil
}

func (s *credentialService) Delete(ctx context.Context, userID, credType string) error {
	if !model.IsValidCredentialType(credType) {
		return errors.NewBadRequestError(fmt.Sprintf("invalid credential type: %s", credType))
	}
	return s.repo.Delete(ctx, userID, credType)
}

func (s *credentialService) List(ctx context.Context, userID string) ([]*CredentialInfo, error) {
	creds, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*CredentialInfo, 0, len(creds))
	for _, c := range creds {
		result = append(result, &CredentialInfo{ID: c.ID.String(), Type: c.Type})
	}
	return result, nil
}

func (s *credentialService) BuildSandboxEnv(ctx context.Context, userID string) (map[string]string, error) {
	creds, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	env := make(map[string]string)
	for _, c := range creds {
		plaintext, err := s.encryptor.Decrypt(c.Encrypted)
		if err != nil {
			slog.Error("failed to decrypt credential for sandbox env",
				"user_id", userID,
				"type", c.Type,
				"error", err)
			continue
		}

		switch c.Type {
		case model.CredentialTypeGitToken:
			env["GIT_TOKEN"] = plaintext
		case model.CredentialTypeDevOpsToken:
			env["DEVOPS_TOKEN"] = plaintext
		}
	}

	return env, nil
}
