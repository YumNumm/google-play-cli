package publisher

import (
	"context"
	"fmt"
	"io"

	androidpublisher "google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"
)

// Client abstracts the Google Play Android Publisher API operations.
type Client interface {
	InsertEdit(ctx context.Context, packageName string) (editID string, err error)
	CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) error
	DeleteEdit(ctx context.Context, packageName, editID string) error
	UploadBundle(ctx context.Context, packageName, editID string, bundle io.Reader) (versionCode int64, err error)
	UpdateTrack(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error
	ListTracks(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error)
}

type googleClient struct {
	service *androidpublisher.Service
}

func NewClient(service *androidpublisher.Service) Client {
	return &googleClient{service: service}
}

func (c *googleClient) InsertEdit(ctx context.Context, packageName string) (string, error) {
	edit, err := c.service.Edits.Insert(packageName, &androidpublisher.AppEdit{}).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to insert edit: %w", err)
	}
	return edit.Id, nil
}

func (c *googleClient) CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview bool) error {
	call := c.service.Edits.Commit(packageName, editID).Context(ctx)
	if changesNotSentForReview {
		call = call.ChangesNotSentForReview(true)
	}
	_, err := call.Do()
	if err != nil {
		return fmt.Errorf("failed to commit edit: %w", err)
	}
	return nil
}

func (c *googleClient) DeleteEdit(ctx context.Context, packageName, editID string) error {
	err := c.service.Edits.Delete(packageName, editID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete edit: %w", err)
	}
	return nil
}

func (c *googleClient) UploadBundle(ctx context.Context, packageName, editID string, bundle io.Reader) (int64, error) {
	b, err := c.service.Edits.Bundles.Upload(packageName, editID).Media(bundle, googleapi.ContentType("application/octet-stream")).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to upload bundle: %w", err)
	}
	return b.VersionCode, nil
}

func (c *googleClient) UpdateTrack(ctx context.Context, packageName, editID, trackName string, track *androidpublisher.Track) error {
	_, err := c.service.Edits.Tracks.Update(packageName, editID, trackName, track).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}
	return nil
}

func (c *googleClient) ListTracks(ctx context.Context, packageName, editID string) ([]*androidpublisher.Track, error) {
	resp, err := c.service.Edits.Tracks.List(packageName, editID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", err)
	}
	return resp.Tracks, nil
}
