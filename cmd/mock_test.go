package cmd

import (
	"context"
	"io"

	androidpublisher "google.golang.org/api/androidpublisher/v3"

	"github.com/YumNumm/google-play-cli/internal/publisher"
)

// mockClient implements publisher.Client for testing.
type mockClient struct {
	insertEditFn   func(ctx context.Context, packageName string) (string, error)
	commitEditFn   func(ctx context.Context, packageName, editID string, changesNotSentForReview bool) error
	deleteEditFn   func(ctx context.Context, packageName, editID string) error
	uploadBundleFn func(ctx context.Context, packageName, editID string, bundle io.Reader) (int64, error)
	updateTrackFn  func(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error
	listTracksFn   func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error)
}

func (m *mockClient) InsertEdit(ctx context.Context, packageName string) (string, error) {
	return m.insertEditFn(ctx, packageName)
}

func (m *mockClient) CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) error {
	return m.commitEditFn(ctx, packageName, editID, changesNotSentForReview)
}

func (m *mockClient) DeleteEdit(ctx context.Context, packageName, editID string) error {
	return m.deleteEditFn(ctx, packageName, editID)
}

func (m *mockClient) UploadBundle(ctx context.Context, packageName, editID string, bundle io.Reader) (int64, error) {
	return m.uploadBundleFn(ctx, packageName, editID, bundle)
}

func (m *mockClient) UpdateTrack(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
	return m.updateTrackFn(ctx, packageName, editID, trackName, track)
}

func (m *mockClient) ListTracks(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
	return m.listTracksFn(ctx, packageName, editID)
}

func successClient() *mockClient {
	return &mockClient{
		insertEditFn: func(ctx context.Context, packageName string) (string, error) {
			return "edit-123", nil
		},
		commitEditFn: func(ctx context.Context, packageName, editID string, changesNotSentForReview bool) error {
			return nil
		},
		deleteEditFn: func(ctx context.Context, packageName, editID string) error {
			return nil
		},
		uploadBundleFn: func(ctx context.Context, packageName, editID string, bundle io.Reader) (int64, error) {
			return 42, nil
		},
		updateTrackFn: func(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
			return nil
		},
		listTracksFn: func(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
			return []*androidpublisher.Track{
				{
					Track: "production",
					Releases: []*androidpublisher.TrackRelease{
						{VersionCodes: []int64{100}, Status: "completed"},
					},
				},
			}, nil
		},
	}
}

func mockFactory(client publisher.Client, err error) ClientFactory {
	return func(ctx context.Context, credentials string) (publisher.Client, error) {
		return client, err
	}
}
