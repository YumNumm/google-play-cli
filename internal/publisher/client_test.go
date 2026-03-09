package publisher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	androidpublisher "google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

func setupTestClient(t *testing.T, handler http.Handler) Client {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	service, err := androidpublisher.NewService(
		context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	return NewClient(service)
}

func TestNewClient(t *testing.T) {
	service, err := androidpublisher.NewService(
		context.Background(),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	client := NewClient(service)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestInsertEdit_Success(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "edit-123"})
	}))

	editID, err := client.InsertEdit(context.Background(), "com.example.app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if editID != "edit-123" {
		t.Fatalf("expected edit-123, got %s", editID)
	}
}

func TestInsertEdit_Error(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": 500, "message": "internal error"}})
	}))

	_, err := client.InsertEdit(context.Background(), "com.example.app")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to insert edit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommitEdit_Success(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "edit-123"})
	}))

	err := client.CommitEdit(context.Background(), "com.example.app", "edit-123", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommitEdit_WithChangesNotSentForReview(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "changesNotSentForReview=true") {
			t.Error("expected changesNotSentForReview=true in query")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "edit-123"})
	}))

	err := client.CommitEdit(context.Background(), "com.example.app", "edit-123", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommitEdit_Error(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": 403, "message": "forbidden"}})
	}))

	err := client.CommitEdit(context.Background(), "com.example.app", "edit-123", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to commit edit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteEdit_Success(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	err := client.DeleteEdit(context.Background(), "com.example.app", "edit-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteEdit_Error(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": 404, "message": "not found"}})
	}))

	err := client.DeleteEdit(context.Background(), "com.example.app", "edit-123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete edit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadBundle_Success(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		_ = body
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"versionCode": 42,
			"sha1":        "abc",
			"sha256":      "def",
		})
	}))

	versionCode, err := client.UploadBundle(context.Background(), "com.example.app", "edit-123", strings.NewReader("fake-aab-data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if versionCode != 42 {
		t.Fatalf("expected version code 42, got %d", versionCode)
	}
}

func TestUploadBundle_Error(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": 400, "message": "bad request"}})
	}))

	_, err := client.UploadBundle(context.Background(), "com.example.app", "edit-123", strings.NewReader("fake"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to upload bundle") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTrack_Success(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"track": "internal",
			"releases": []map[string]any{
				{"status": "completed", "versionCodes": []string{"42"}},
			},
		})
	}))

	track := &androidpublisher.Track{
		Track: "internal",
		Releases: []*androidpublisher.TrackRelease{
			{Status: "completed", VersionCodes: []int64{42}},
		},
	}
	err := client.UpdateTrack(context.Background(), "com.example.app", "edit-123", "internal", track)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTrack_Error(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": 409, "message": "conflict"}})
	}))

	err := client.UpdateTrack(context.Background(), "com.example.app", "edit-123", "internal", &androidpublisher.Track{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to update track") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListTracks_Success(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"kind": "androidpublisher#tracksListResponse",
			"tracks": []map[string]any{
				{
					"track": "production",
					"releases": []map[string]any{
						{"versionCodes": []string{"100"}, "status": "completed"},
					},
				},
				{
					"track": "beta",
					"releases": []map[string]any{
						{"versionCodes": []string{"120"}, "status": "completed"},
					},
				},
			},
		})
	}))

	tracks, err := client.ListTracks(context.Background(), "com.example.app", "edit-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(tracks))
	}
	if tracks[0].Track != "production" {
		t.Fatalf("expected production, got %s", tracks[0].Track)
	}
}

func TestListTracks_Error(t *testing.T) {
	client := setupTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": 401, "message": "unauthorized"}})
	}))

	_, err := client.ListTracks(context.Background(), "com.example.app", "edit-123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to list tracks") {
		t.Fatalf("unexpected error: %v", err)
	}
}
