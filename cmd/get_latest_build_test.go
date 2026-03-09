package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	androidpublisher "google.golang.org/api/androidpublisher/v3"

	"github.com/YumNumm/google-play-cli/internal/publisher"
)

func TestRunGetLatestBuildNumber_Success(t *testing.T) {
	client := successClient()
	var buf bytes.Buffer

	opts := getLatestBuildOptions{
		packageName: "com.example.app",
	}

	err := runGetLatestBuildNumber(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "100" {
		t.Fatalf("expected 100, got %s", buf.String())
	}
}

func TestRunGetLatestBuildNumber_WithTrackFilter(t *testing.T) {
	client := successClient()
	client.listTracksFn = func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
		return []*androidpublisher.Track{
			{
				Track: "production",
				Releases: []*androidpublisher.TrackRelease{
					{VersionCodes: []int64{100}, Status: "completed"},
				},
			},
			{
				Track: "beta",
				Releases: []*androidpublisher.TrackRelease{
					{VersionCodes: []int64{200}, Status: "completed"},
				},
			},
			{
				Track: "alpha",
				Releases: []*androidpublisher.TrackRelease{
					{VersionCodes: []int64{300}, Status: "completed"},
				},
			},
		}, nil
	}
	var buf bytes.Buffer

	opts := getLatestBuildOptions{
		packageName: "com.example.app",
		tracks:      []string{"production", "beta"},
	}

	err := runGetLatestBuildNumber(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "200" {
		t.Fatalf("expected 200, got %s", buf.String())
	}
}

func TestRunGetLatestBuildNumber_MultipleReleasesAndVersionCodes(t *testing.T) {
	client := successClient()
	client.listTracksFn = func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
		return []*androidpublisher.Track{
			{
				Track: "production",
				Releases: []*androidpublisher.TrackRelease{
					{VersionCodes: []int64{100, 101}, Status: "completed"},
					{VersionCodes: []int64{110}, Status: "draft"},
				},
			},
			{
				Track: "beta",
				Releases: []*androidpublisher.TrackRelease{
					{VersionCodes: []int64{120}, Status: "completed"},
				},
			},
		}, nil
	}
	var buf bytes.Buffer

	opts := getLatestBuildOptions{
		packageName: "com.example.app",
	}

	err := runGetLatestBuildNumber(context.Background(), client, opts, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "120" {
		t.Fatalf("expected 120, got %s", buf.String())
	}
}

func TestRunGetLatestBuildNumber_InsertEditError(t *testing.T) {
	client := successClient()
	client.insertEditFn = func(ctx context.Context, packageName string) (string, error) {
		return "", errors.New("insert failed")
	}

	opts := getLatestBuildOptions{packageName: "com.example.app"}

	err := runGetLatestBuildNumber(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "insert failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGetLatestBuildNumber_ListTracksError(t *testing.T) {
	client := successClient()
	client.listTracksFn = func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
		return nil, errors.New("list failed")
	}

	opts := getLatestBuildOptions{packageName: "com.example.app"}

	err := runGetLatestBuildNumber(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "list failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGetLatestBuildNumber_DeleteEditError(t *testing.T) {
	client := successClient()
	client.deleteEditFn = func(ctx context.Context, packageName, editID string) error {
		return errors.New("delete failed")
	}

	opts := getLatestBuildOptions{packageName: "com.example.app"}

	err := runGetLatestBuildNumber(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGetLatestBuildNumber_ListErrorTakesPriority(t *testing.T) {
	client := successClient()
	client.listTracksFn = func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
		return nil, errors.New("list failed")
	}
	client.deleteEditFn = func(ctx context.Context, packageName, editID string) error {
		return errors.New("delete also failed")
	}

	opts := getLatestBuildOptions{packageName: "com.example.app"}

	err := runGetLatestBuildNumber(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "list failed") {
		t.Fatalf("expected list error to take priority, got: %v", err)
	}
}

func TestRunGetLatestBuildNumber_NoVersionCodes(t *testing.T) {
	client := successClient()
	client.listTracksFn = func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
		return []*androidpublisher.Track{
			{Track: "production", Releases: []*androidpublisher.TrackRelease{}},
		}, nil
	}

	opts := getLatestBuildOptions{packageName: "com.example.app"}

	err := runGetLatestBuildNumber(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no version codes found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGetLatestBuildNumber_EmptyTracksAfterFilter(t *testing.T) {
	client := successClient()
	client.listTracksFn = func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
		return []*androidpublisher.Track{
			{
				Track: "production",
				Releases: []*androidpublisher.TrackRelease{
					{VersionCodes: []int64{100}},
				},
			},
		}, nil
	}

	opts := getLatestBuildOptions{
		packageName: "com.example.app",
		tracks:      []string{"beta"},
	}

	err := runGetLatestBuildNumber(context.Background(), client, opts, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no version codes found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterTracks(t *testing.T) {
	tracks := []*androidpublisher.Track{
		{Track: "production"},
		{Track: "beta"},
		{Track: "alpha"},
	}

	filtered := filterTracks(tracks, []string{"beta", "alpha"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(filtered))
	}
	if filtered[0].Track != "beta" {
		t.Fatalf("expected beta, got %s", filtered[0].Track)
	}
	if filtered[1].Track != "alpha" {
		t.Fatalf("expected alpha, got %s", filtered[1].Track)
	}
}

func TestFilterTracks_NoMatch(t *testing.T) {
	tracks := []*androidpublisher.Track{
		{Track: "production"},
	}

	filtered := filterTracks(tracks, []string{"beta"})
	if len(filtered) != 0 {
		t.Fatalf("expected 0 tracks, got %d", len(filtered))
	}
}

func TestFindMaxVersionCode(t *testing.T) {
	tracks := []*androidpublisher.Track{
		{
			Track: "production",
			Releases: []*androidpublisher.TrackRelease{
				{VersionCodes: []int64{100, 101}},
				{VersionCodes: []int64{50}},
			},
		},
		{
			Track: "beta",
			Releases: []*androidpublisher.TrackRelease{
				{VersionCodes: []int64{200}},
			},
		},
	}

	max := findMaxVersionCode(tracks)
	if max != 200 {
		t.Fatalf("expected 200, got %d", max)
	}
}

func TestFindMaxVersionCode_EmptyTracks(t *testing.T) {
	max := findMaxVersionCode(nil)
	if max != -1 {
		t.Fatalf("expected -1, got %d", max)
	}
}

func TestFindMaxVersionCode_NoReleases(t *testing.T) {
	tracks := []*androidpublisher.Track{
		{Track: "production", Releases: nil},
	}
	max := findMaxVersionCode(tracks)
	if max != -1 {
		t.Fatalf("expected -1, got %d", max)
	}
}

func TestFindMaxVersionCode_EmptyVersionCodes(t *testing.T) {
	tracks := []*androidpublisher.Track{
		{
			Track: "production",
			Releases: []*androidpublisher.TrackRelease{
				{VersionCodes: nil},
			},
		},
	}
	max := findMaxVersionCode(tracks)
	if max != -1 {
		t.Fatalf("expected -1, got %d", max)
	}
}

func TestGetLatestBuildNumberCmd_FactoryError(t *testing.T) {
	factory := mockFactory(nil, errors.New("auth failed"))
	root := newRootCmd(factory)
	root.SetArgs([]string{"get-latest-build-number", "-p", "com.example.app"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetLatestBuildNumberCmd_MissingPackageName(t *testing.T) {
	factory := mockFactory(successClient(), nil)
	root := newRootCmd(factory)
	root.SetArgs([]string{"get-latest-build-number"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing required flag")
	}
}

func TestGetLatestBuildNumberCmd_Integration(t *testing.T) {
	client := successClient()
	client.listTracksFn = func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
		return []*androidpublisher.Track{
			{
				Track: "production",
				Releases: []*androidpublisher.TrackRelease{
					{VersionCodes: []int64{42}, Status: "completed"},
				},
			},
		}, nil
	}

	factory := func(ctx context.Context, credentials string) (publisher.Client, error) {
		return client, nil
	}
	root := newRootCmd(factory)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"get-latest-build-number", "-p", "com.example.app"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "42" {
		t.Fatalf("expected 42, got %s", buf.String())
	}
}

func TestGetLatestBuildNumberCmd_WithTracksFlag(t *testing.T) {
	client := successClient()
	client.listTracksFn = func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
		return []*androidpublisher.Track{
			{Track: "production", Releases: []*androidpublisher.TrackRelease{{VersionCodes: []int64{100}}}},
			{Track: "beta", Releases: []*androidpublisher.TrackRelease{{VersionCodes: []int64{200}}}},
		}, nil
	}
	factory := func(ctx context.Context, credentials string) (publisher.Client, error) {
		return client, nil
	}
	root := newRootCmd(factory)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"get-latest-build-number", "-p", "com.example.app", "-t", "production"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "100" {
		t.Fatalf("expected 100, got %s", buf.String())
	}
}
