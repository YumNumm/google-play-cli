package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	androidpublisher "google.golang.org/api/androidpublisher/v3"

	"github.com/YumNumm/google-play-cli/internal/publisher"
)

type bundlesPublishOptions struct {
	packageName             string
	bundlePath              string
	track                   string
	credentials             string
	releaseName             string
	releaseNotes            string
	inAppUpdatePriority     int
	rolloutFraction         float64
	draft                   bool
	changesNotSentForReview bool
}

func newBundlesPublishCmd(factory ClientFactory) *cobra.Command {
	var opts bundlesPublishOptions

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Upload an AAB and publish to a track",
		Long:  "Upload an Android App Bundle (AAB) to Google Play and publish it to the specified track.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := factory(ctx, opts.credentials)
			if err != nil {
				return err
			}
			return runBundlesPublish(ctx, client, opts, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&opts.packageName, "package-name", "p", "", "Application package name (required)")
	cmd.Flags().StringVarP(&opts.bundlePath, "bundle", "b", "", "Path to the AAB file (required)")
	cmd.Flags().StringVarP(&opts.track, "track", "t", "", "Track name: production, beta, alpha, internal (required)")
	cmd.Flags().StringVar(&opts.credentials, "credentials", "", "Service account JSON string (omit to use ADC/WIF)")
	cmd.Flags().StringVarP(&opts.releaseName, "release-name", "r", "", "Release name")
	cmd.Flags().StringVarP(&opts.releaseNotes, "release-notes", "n", "", `Release notes as JSON: [{"language":"en-US","text":"Bug fixes"}]`)
	cmd.Flags().IntVarP(&opts.inAppUpdatePriority, "in-app-update-priority", "i", 0, "In-app update priority (0-5)")
	cmd.Flags().Float64VarP(&opts.rolloutFraction, "rollout-fraction", "f", 0, "Staged rollout fraction (0.0-1.0)")
	cmd.Flags().BoolVarP(&opts.draft, "draft", "d", false, "Set release as draft")
	cmd.Flags().BoolVar(&opts.changesNotSentForReview, "changes-not-sent-for-review", false, "Do not send changes for review")

	_ = cmd.MarkFlagRequired("package-name")
	_ = cmd.MarkFlagRequired("bundle")
	_ = cmd.MarkFlagRequired("track")

	return cmd
}

const uploadTimeout = 10 * time.Minute

func runBundlesPublish(ctx context.Context, client publisher.Client, opts bundlesPublishOptions, stdout io.Writer) error {
	if err := validatePublishOptions(opts); err != nil {
		return err
	}

	var releaseNotes []*androidpublisher.LocalizedText
	if opts.releaseNotes != "" {
		var err error
		releaseNotes, err = parseReleaseNotes(opts.releaseNotes)
		if err != nil {
			return err
		}
	}

	editID, err := client.InsertEdit(ctx, opts.packageName)
	if err != nil {
		return err
	}

	bundleFile, err := os.Open(opts.bundlePath)
	if err != nil {
		return fmt.Errorf("failed to open bundle file %q: %w", opts.bundlePath, err)
	}
	defer bundleFile.Close()

	uploadCtx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	versionCode, err := client.UploadBundle(uploadCtx, opts.packageName, editID, bundleFile)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Uploaded bundle with version code: %d\n", versionCode)

	release := buildRelease(opts, versionCode, releaseNotes)
	track := &androidpublisher.Track{
		Track:    opts.track,
		Releases: []*androidpublisher.TrackRelease{release},
	}

	if err := client.UpdateTrack(ctx, opts.packageName, editID, opts.track, track); err != nil {
		return err
	}

	if err := client.CommitEdit(ctx, opts.packageName, editID, opts.changesNotSentForReview); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Successfully published to track %q\n", opts.track)
	return nil
}

func buildRelease(opts bundlesPublishOptions, versionCode int64, releaseNotes []*androidpublisher.LocalizedText) *androidpublisher.TrackRelease {
	release := &androidpublisher.TrackRelease{
		VersionCodes: []int64{versionCode},
		ReleaseNotes: releaseNotes,
	}

	if opts.releaseName != "" {
		release.Name = opts.releaseName
	}

	if opts.inAppUpdatePriority > 0 {
		release.InAppUpdatePriority = int64(opts.inAppUpdatePriority)
	}

	switch {
	case opts.draft:
		release.Status = "draft"
	case opts.rolloutFraction > 0:
		release.Status = "inProgress"
		release.UserFraction = opts.rolloutFraction
	default:
		release.Status = "completed"
	}

	return release
}

func validatePublishOptions(opts bundlesPublishOptions) error {
	if opts.draft && opts.rolloutFraction > 0 {
		return fmt.Errorf("--draft and --rollout-fraction are mutually exclusive")
	}
	if opts.inAppUpdatePriority < 0 || opts.inAppUpdatePriority > 5 {
		return fmt.Errorf("--in-app-update-priority must be between 0 and 5, got: %d", opts.inAppUpdatePriority)
	}
	if opts.rolloutFraction < 0 || opts.rolloutFraction > 1 {
		return fmt.Errorf("--rollout-fraction must be between 0.0 and 1.0, got: %f", opts.rolloutFraction)
	}
	return nil
}

func parseReleaseNotes(jsonStr string) ([]*androidpublisher.LocalizedText, error) {
	var notes []*androidpublisher.LocalizedText
	if err := json.Unmarshal([]byte(jsonStr), &notes); err != nil {
		return nil, fmt.Errorf("failed to parse --release-notes JSON: %w\nExpected format: [{\"language\":\"en-US\",\"text\":\"Release notes\"}]", err)
	}
	if len(notes) == 0 {
		return nil, fmt.Errorf("--release-notes must contain at least one entry")
	}
	for i, note := range notes {
		if note.Language == "" {
			return nil, fmt.Errorf("--release-notes entry %d: language must not be empty", i)
		}
	}
	return notes, nil
}
