package auth

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func dummyCreds() *google.Credentials {
	return &google.Credentials{
		TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
	}
}

func TestNewResolver(t *testing.T) {
	r := NewResolver()
	if r == nil {
		t.Fatal("expected non-nil resolver")
	}
	if r.credentialsFromJSON == nil {
		t.Fatal("expected credentialsFromJSON to be set")
	}
	if r.findDefaultCredentials == nil {
		t.Fatal("expected findDefaultCredentials to be set")
	}
}

func TestResolve_WithValidServiceAccountJSON(t *testing.T) {
	expected := dummyCreds()
	r := &CredentialResolver{
		credentialsFromJSON: func(ctx context.Context, jsonData []byte, scopes ...string) (*google.Credentials, error) {
			return expected, nil
		},
	}

	creds, err := r.Resolve(context.Background(), `{"type":"service_account","project_id":"test"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds != expected {
		t.Fatal("expected returned credentials to match")
	}
}

func TestResolve_WithInvalidJSON(t *testing.T) {
	r := &CredentialResolver{
		credentialsFromJSON: func(ctx context.Context, jsonData []byte, scopes ...string) (*google.Credentials, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}

	_, err := r.Resolve(context.Background(), "not-json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if got := err.Error(); !contains(got, "failed to parse credentials JSON") {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestResolve_WithWrongType(t *testing.T) {
	r := &CredentialResolver{
		credentialsFromJSON: func(ctx context.Context, jsonData []byte, scopes ...string) (*google.Credentials, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}

	_, err := r.Resolve(context.Background(), `{"type":"external_account"}`)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if got := err.Error(); !contains(got, "only supports service_account type") {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestResolve_WithEmptyType(t *testing.T) {
	r := &CredentialResolver{
		credentialsFromJSON: func(ctx context.Context, jsonData []byte, scopes ...string) (*google.Credentials, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}

	_, err := r.Resolve(context.Background(), `{"project_id":"test"}`)
	if err == nil {
		t.Fatal("expected error for missing type")
	}
	if got := err.Error(); !contains(got, "only supports service_account type") {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestResolve_CredentialsFromJSONError(t *testing.T) {
	r := &CredentialResolver{
		credentialsFromJSON: func(ctx context.Context, jsonData []byte, scopes ...string) (*google.Credentials, error) {
			return nil, errors.New("mock creds error")
		},
	}

	_, err := r.Resolve(context.Background(), `{"type":"service_account"}`)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !contains(got, "failed to create credentials from JSON") {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestResolve_FallbackToADC_Success(t *testing.T) {
	expected := dummyCreds()
	r := &CredentialResolver{
		findDefaultCredentials: func(ctx context.Context, scopes ...string) (*google.Credentials, error) {
			return expected, nil
		},
	}

	creds, err := r.Resolve(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds != expected {
		t.Fatal("expected returned credentials to match")
	}
}

func TestResolve_FallbackToADC_Error(t *testing.T) {
	r := &CredentialResolver{
		findDefaultCredentials: func(ctx context.Context, scopes ...string) (*google.Credentials, error) {
			return nil, errors.New("no creds")
		},
	}

	_, err := r.Resolve(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !contains(got, "no credentials found") {
		t.Fatalf("unexpected error message: %s", got)
	}
	if got := err.Error(); !contains(got, "GOOGLE_APPLICATION_CREDENTIALS") {
		t.Fatalf("expected hint about env var in error: %s", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
