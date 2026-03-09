package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"testing"

	androidpublisher "google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

func TestNewRootCmd(t *testing.T) {
	factory := mockFactory(successClient(), nil)
	root := newRootCmd(factory)

	if root == nil {
		t.Fatal("expected non-nil root command")
	}
	if root.Use != "google-play-cli" {
		t.Fatalf("expected use 'google-play-cli', got %s", root.Use)
	}
}

func TestNewRootCmd_HasBundlesSubcommand(t *testing.T) {
	factory := mockFactory(successClient(), nil)
	root := newRootCmd(factory)

	bundlesCmd, _, err := root.Find([]string{"bundles"})
	if err != nil {
		t.Fatalf("unexpected error finding bundles: %v", err)
	}
	if bundlesCmd.Use != "bundles" {
		t.Fatalf("expected bundles command, got %s", bundlesCmd.Use)
	}
}

func TestNewRootCmd_HasGetLatestBuildNumberSubcommand(t *testing.T) {
	factory := mockFactory(successClient(), nil)
	root := newRootCmd(factory)

	cmd, _, err := root.Find([]string{"get-latest-build-number"})
	if err != nil {
		t.Fatalf("unexpected error finding get-latest-build-number: %v", err)
	}
	if cmd.Use != "get-latest-build-number" {
		t.Fatalf("expected get-latest-build-number command, got %s", cmd.Use)
	}
}

func TestRootCmd_NoArgs_ShowsHelp(t *testing.T) {
	factory := mockFactory(successClient(), nil)
	root := newRootCmd(factory)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{})

	_ = root.Execute()
	if buf.Len() == 0 {
		t.Fatal("expected help output")
	}
}

func TestRootCmd_SilenceUsage(t *testing.T) {
	factory := mockFactory(successClient(), nil)
	root := newRootCmd(factory)
	if !root.SilenceUsage {
		t.Fatal("expected SilenceUsage to be true")
	}
}

func TestExecute(t *testing.T) {
	// Execute with no args shows help and returns nil
	// We can't fully test DefaultClientFactory here without real credentials,
	// but this covers the Execute function itself.
	// Note: Execute() creates a new root command each time.
	err := Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultClientFactory_ServiceCreationError(t *testing.T) {
	orig := createPublisherService
	defer func() { createPublisherService = orig }()
	createPublisherService = func(ctx context.Context, opts ...option.ClientOption) (*androidpublisher.Service, error) {
		return nil, errors.New("service creation failed")
	}

	privateKey := generateTestPrivateKey(t)
	saJSON := fmt.Sprintf(`{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "key-id",
		"private_key": %q,
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "123456789",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token"
	}`, privateKey)

	_, err := DefaultClientFactory(context.Background(), saJSON)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create publisher service") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func generateTestPrivateKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(block))
}

func TestDefaultClientFactory_WithValidSAJSON(t *testing.T) {
	privateKey := generateTestPrivateKey(t)
	saJSON := fmt.Sprintf(`{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "key-id",
		"private_key": %q,
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "123456789",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token"
	}`, privateKey)

	client, err := DefaultClientFactory(context.Background(), saJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestDefaultClientFactory_WithInvalidJSON(t *testing.T) {
	_, err := DefaultClientFactory(context.Background(), "not-json")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultClientFactory_WithWrongType(t *testing.T) {
	_, err := DefaultClientFactory(context.Background(), `{"type":"external_account"}`)
	if err == nil {
		t.Fatal("expected error")
	}
}
