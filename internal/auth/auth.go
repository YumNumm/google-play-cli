package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2/google"
	androidpublisher "google.golang.org/api/androidpublisher/v3"
)

var scope = androidpublisher.AndroidpublisherScope

// CredentialResolver resolves Google Cloud credentials from either an explicit
// service account JSON string or Application Default Credentials (ADC).
type CredentialResolver struct {
	credentialsFromJSON    func(ctx context.Context, jsonData []byte, scope ...string) (*google.Credentials, error)
	findDefaultCredentials func(ctx context.Context, scopes ...string) (*google.Credentials, error)
}

func NewResolver() *CredentialResolver {
	return &CredentialResolver{
		credentialsFromJSON:    google.CredentialsFromJSON,
		findDefaultCredentials: google.FindDefaultCredentials,
	}
}

// Resolve returns Google credentials. If credentialsJSON is non-empty, it is
// parsed as a service account key JSON string. Otherwise, Application Default
// Credentials are used (supporting WIF via GOOGLE_APPLICATION_CREDENTIALS).
func (r *CredentialResolver) Resolve(ctx context.Context, credentialsJSON string) (*google.Credentials, error) {
	if credentialsJSON != "" {
		return r.fromServiceAccountJSON(ctx, credentialsJSON)
	}
	return r.fromDefaultCredentials(ctx)
}

func (r *CredentialResolver) fromServiceAccountJSON(ctx context.Context, jsonStr string) (*google.Credentials, error) {
	var parsed struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse credentials JSON: %w", err)
	}
	if parsed.Type != "service_account" {
		return nil, fmt.Errorf("--credentials only supports service_account type, got: %q", parsed.Type)
	}
	creds, err := r.credentialsFromJSON(ctx, []byte(jsonStr), scope)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials from JSON: %w", err)
	}
	return creds, nil
}

func (r *CredentialResolver) fromDefaultCredentials(ctx context.Context) (*google.Credentials, error) {
	creds, err := r.findDefaultCredentials(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("no credentials found: set --credentials flag or GOOGLE_APPLICATION_CREDENTIALS environment variable: %w", err)
	}
	return creds, nil
}
