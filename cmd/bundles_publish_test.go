package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	androidpublisher "google.golang.org/api/androidpublisher/v3"
)

func createTempAAB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "app.aab")
	if err := os.WriteFile(path, []byte("fake-aab-content"), 0644); err != nil {
		t.Fatalf("failed to create temp AAB: %v", err)
	}
	return path
}

func TestRunBundlesPublish_Success(t *testing.T) {
	client := successClient()
	aabPath := createTempAAB(t)
	var buf bytes.Buffer

	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  aabPath,
		track:       "internal",
	}

	err := runBundlesPublish(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "version code: 42") {
		t.Fatalf("expected version code in output, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), `track "internal"`) {
		t.Fatalf("expected track name in output, got: %s", buf.String())
	}
}

func TestRunBundlesPublish_WithReleaseName(t *testing.T) {
	var capturedTrack *androidpublisher.Track
	client := successClient()
	client.updateTrackFn = func(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
		capturedTrack = track
		return nil
	}
	aabPath := createTempAAB(t)
	var buf bytes.Buffer

	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  aabPath,
		track:       "internal",
		releaseName: "v1.0.0",
	}

	err := runBundlesPublish(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTrack.Releases[0].Name != "v1.0.0" {
		t.Fatalf("expected release name v1.0.0, got %s", capturedTrack.Releases[0].Name)
	}
}

func TestRunBundlesPublish_WithReleaseNotes(t *testing.T) {
	var capturedTrack *androidpublisher.Track
	client := successClient()
	client.updateTrackFn = func(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
		capturedTrack = track
		return nil
	}
	aabPath := createTempAAB(t)
	var buf bytes.Buffer

	opts := bundlesPublishOptions{
		packageName:  "com.example.app",
		bundlePath:   aabPath,
		track:        "internal",
		releaseNotes: `[{"language":"en-US","text":"Bug fixes"}]`,
	}

	err := runBundlesPublish(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedTrack.Releases[0].ReleaseNotes) != 1 {
		t.Fatal("expected 1 release note")
	}
	if capturedTrack.Releases[0].ReleaseNotes[0].Language != "en-US" {
		t.Fatal("expected en-US language")
	}
}

func TestRunBundlesPublish_DraftMode(t *testing.T) {
	var capturedTrack *androidpublisher.Track
	client := successClient()
	client.updateTrackFn = func(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
		capturedTrack = track
		return nil
	}
	aabPath := createTempAAB(t)
	var buf bytes.Buffer

	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  aabPath,
		track:       "internal",
		draft:       true,
	}

	err := runBundlesPublish(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTrack.Releases[0].Status != "draft" {
		t.Fatalf("expected draft status, got %s", capturedTrack.Releases[0].Status)
	}
}

func TestRunBundlesPublish_RolloutFraction(t *testing.T) {
	var capturedTrack *androidpublisher.Track
	client := successClient()
	client.updateTrackFn = func(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
		capturedTrack = track
		return nil
	}
	aabPath := createTempAAB(t)
	var buf bytes.Buffer

	opts := bundlesPublishOptions{
		packageName:     "com.example.app",
		bundlePath:      aabPath,
		track:           "production",
		rolloutFraction: 0.5,
	}

	err := runBundlesPublish(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTrack.Releases[0].Status != "inProgress" {
		t.Fatalf("expected inProgress status, got %s", capturedTrack.Releases[0].Status)
	}
	if capturedTrack.Releases[0].UserFraction != 0.5 {
		t.Fatalf("expected 0.5 user fraction, got %f", capturedTrack.Releases[0].UserFraction)
	}
}

func TestRunBundlesPublish_InAppUpdatePriority(t *testing.T) {
	var capturedTrack *androidpublisher.Track
	client := successClient()
	client.updateTrackFn = func(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
		capturedTrack = track
		return nil
	}
	aabPath := createTempAAB(t)
	var buf bytes.Buffer

	opts := bundlesPublishOptions{
		packageName:         "com.example.app",
		bundlePath:          aabPath,
		track:               "internal",
		inAppUpdatePriority: 3,
	}

	err := runBundlesPublish(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTrack.Releases[0].InAppUpdatePriority != 3 {
		t.Fatalf("expected priority 3, got %d", capturedTrack.Releases[0].InAppUpdatePriority)
	}
}

func TestRunBundlesPublish_ChangesNotSentForReview(t *testing.T) {
	var capturedChangesNotSent bool
	client := successClient()
	client.commitEditFn = func(ctx context.Context, packageName, editID string, changesNotSentForReview bool) error {
		capturedChangesNotSent = changesNotSentForReview
		return nil
	}
	aabPath := createTempAAB(t)
	var buf bytes.Buffer

	opts := bundlesPublishOptions{
		packageName:             "com.example.app",
		bundlePath:              aabPath,
		track:                   "internal",
		changesNotSentForReview: true,
	}

	err := runBundlesPublish(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !capturedChangesNotSent {
		t.Fatal("expected changesNotSentForReview to be true")
	}
}

func TestRunBundlesPublish_ValidationError_DraftAndRollout(t *testing.T) {
	opts := bundlesPublishOptions{
		packageName:     "com.example.app",
		bundlePath:      "app.aab",
		track:           "internal",
		draft:           true,
		rolloutFraction: 0.5,
	}

	err := runBundlesPublish(context.Background(), successClient(), opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_ValidationError_PriorityTooHigh(t *testing.T) {
	opts := bundlesPublishOptions{
		packageName:         "com.example.app",
		bundlePath:          "app.aab",
		track:               "internal",
		inAppUpdatePriority: 6,
	}

	err := runBundlesPublish(context.Background(), successClient(), opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "between 0 and 5") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_ValidationError_PriorityNegative(t *testing.T) {
	opts := bundlesPublishOptions{
		packageName:         "com.example.app",
		bundlePath:          "app.aab",
		track:               "internal",
		inAppUpdatePriority: -1,
	}

	err := runBundlesPublish(context.Background(), successClient(), opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "between 0 and 5") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_ValidationError_RolloutTooHigh(t *testing.T) {
	opts := bundlesPublishOptions{
		packageName:     "com.example.app",
		bundlePath:      "app.aab",
		track:           "internal",
		rolloutFraction: 1.5,
	}

	err := runBundlesPublish(context.Background(), successClient(), opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "between 0.0 and 1.0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_ValidationError_RolloutNegative(t *testing.T) {
	opts := bundlesPublishOptions{
		packageName:     "com.example.app",
		bundlePath:      "app.aab",
		track:           "internal",
		rolloutFraction: -0.1,
	}

	err := runBundlesPublish(context.Background(), successClient(), opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "between 0.0 and 1.0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_InvalidReleaseNotes(t *testing.T) {
	aabPath := createTempAAB(t)
	opts := bundlesPublishOptions{
		packageName:  "com.example.app",
		bundlePath:   aabPath,
		track:        "internal",
		releaseNotes: "not-json",
	}

	err := runBundlesPublish(context.Background(), successClient(), opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to parse --release-notes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_InsertEditError(t *testing.T) {
	client := successClient()
	client.insertEditFn = func(ctx context.Context, packageName string) (string, error) {
		return "", errors.New("insert failed")
	}
	aabPath := createTempAAB(t)

	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  aabPath,
		track:       "internal",
	}

	err := runBundlesPublish(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "insert failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_BundleFileNotFound(t *testing.T) {
	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  "/nonexistent/path/app.aab",
		track:       "internal",
	}

	err := runBundlesPublish(context.Background(), successClient(), opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to open bundle file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_UploadBundleError(t *testing.T) {
	client := successClient()
	client.uploadBundleFn = func(ctx context.Context, packageName, editID string, bundle io.Reader) (int64, error) {
		return 0, errors.New("upload failed")
	}
	aabPath := createTempAAB(t)

	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  aabPath,
		track:       "internal",
	}

	err := runBundlesPublish(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "upload failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_UpdateTrackError(t *testing.T) {
	client := successClient()
	client.updateTrackFn = func(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
		return errors.New("track update failed")
	}
	aabPath := createTempAAB(t)

	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  aabPath,
		track:       "internal",
	}

	err := runBundlesPublish(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "track update failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBundlesPublish_CommitEditError(t *testing.T) {
	client := successClient()
	client.commitEditFn = func(ctx context.Context, packageName, editID string, changesNotSentForReview bool) error {
		return errors.New("commit failed")
	}
	aabPath := createTempAAB(t)

	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  aabPath,
		track:       "internal",
	}

	err := runBundlesPublish(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "commit failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseReleaseNotes_Valid(t *testing.T) {
	notes, err := parseReleaseNotes(`[{"language":"en-US","text":"Bug fixes"},{"language":"ja-JP","text":"バグ修正"}]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].Language != "en-US" || notes[0].Text != "Bug fixes" {
		t.Fatalf("unexpected first note: %+v", notes[0])
	}
	if notes[1].Language != "ja-JP" || notes[1].Text != "バグ修正" {
		t.Fatalf("unexpected second note: %+v", notes[1])
	}
}

func TestParseReleaseNotes_InvalidJSON(t *testing.T) {
	_, err := parseReleaseNotes("not-json")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "Expected format") {
		t.Fatalf("expected format hint in error: %v", err)
	}
}

func TestParseReleaseNotes_EmptyArray(t *testing.T) {
	_, err := parseReleaseNotes("[]")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "at least one entry") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseReleaseNotes_EmptyLanguage(t *testing.T) {
	_, err := parseReleaseNotes(`[{"language":"","text":"test"}]`)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "language must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePublishOptions_Valid(t *testing.T) {
	opts := bundlesPublishOptions{
		packageName: "com.example.app",
		bundlePath:  "app.aab",
		track:       "internal",
	}
	if err := validatePublishOptions(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBundlesPublishCmd_FactoryError(t *testing.T) {
	factory := mockFactory(nil, errors.New("auth failed"))
	root := newRootCmd(factory)
	root.SetArgs([]string{"bundles", "publish", "-p", "com.example.app", "-b", "app.aab", "-t", "internal"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBundlesPublishCmd_MissingRequiredFlags(t *testing.T) {
	factory := mockFactory(successClient(), nil)
	root := newRootCmd(factory)
	root.SetArgs([]string{"bundles", "publish"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestBundlesPublishCmd_Integration(t *testing.T) {
	aabPath := createTempAAB(t)
	factory := mockFactory(successClient(), nil)
	root := newRootCmd(factory)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{
		"bundles", "publish",
		"-p", "com.example.app",
		"-b", aabPath,
		"-t", "internal",
	})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Successfully published") {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestBuildRelease_Completed(t *testing.T) {
	opts := bundlesPublishOptions{track: "production"}
	release := buildRelease(opts, 42, nil)

	if release.Status != "completed" {
		t.Fatalf("expected completed, got %s", release.Status)
	}
	if len(release.VersionCodes) != 1 || release.VersionCodes[0] != 42 {
		t.Fatalf("unexpected version codes: %v", release.VersionCodes)
	}
}

func TestBuildRelease_Draft(t *testing.T) {
	opts := bundlesPublishOptions{draft: true}
	release := buildRelease(opts, 42, nil)

	if release.Status != "draft" {
		t.Fatalf("expected draft, got %s", release.Status)
	}
}

func TestBuildRelease_Rollout(t *testing.T) {
	opts := bundlesPublishOptions{rolloutFraction: 0.25}
	release := buildRelease(opts, 42, nil)

	if release.Status != "inProgress" {
		t.Fatalf("expected inProgress, got %s", release.Status)
	}
	if release.UserFraction != 0.25 {
		t.Fatalf("expected 0.25, got %f", release.UserFraction)
	}
}

func TestBuildRelease_WithAllOptions(t *testing.T) {
	notes := []*androidpublisher.LocalizedText{{Language: "en-US", Text: "test"}}
	opts := bundlesPublishOptions{
		releaseName:         "v2.0",
		inAppUpdatePriority: 4,
	}
	release := buildRelease(opts, 99, notes)

	if release.Name != "v2.0" {
		t.Fatalf("expected v2.0, got %s", release.Name)
	}
	if release.InAppUpdatePriority != 4 {
		t.Fatalf("expected 4, got %d", release.InAppUpdatePriority)
	}
	if len(release.ReleaseNotes) != 1 {
		t.Fatal("expected 1 release note")
	}
}
