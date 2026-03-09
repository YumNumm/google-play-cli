package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	androidpublisher "google.golang.org/api/androidpublisher/v3"

	"github.com/YumNumm/google-play-cli/internal/publisher"
)

type getLatestBuildOptions struct {
	packageName string
	tracks      []string
	credentials string
}

func newGetLatestBuildNumberCmd(factory ClientFactory) *cobra.Command {
	var opts getLatestBuildOptions

	cmd := &cobra.Command{
		Use:   "get-latest-build-number",
		Short: "Get the latest build number (versionCode) from Google Play",
		Long:  "Retrieve the highest versionCode across all (or specified) tracks for a given package.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := factory(ctx, opts.credentials)
			if err != nil {
				return err
			}
			return runGetLatestBuildNumber(ctx, client, opts, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&opts.packageName, "package-name", "p", "", "Application package name (required)")
	cmd.Flags().StringSliceVarP(&opts.tracks, "tracks", "t", nil, "Filter by track names (comma-separated or repeated)")
	cmd.Flags().StringVar(&opts.credentials, "credentials", "", "Service account JSON string (omit to use ADC/WIF)")

	_ = cmd.MarkFlagRequired("package-name")

	return cmd
}

func runGetLatestBuildNumber(ctx context.Context, client publisher.Client, opts getLatestBuildOptions, stdout io.Writer) error {
	editID, err := client.InsertEdit(ctx, opts.packageName)
	if err != nil {
		return err
	}

	tracks, listErr := client.ListTracks(ctx, opts.packageName, editID)
	deleteErr := client.DeleteEdit(ctx, opts.packageName, editID)

	if listErr != nil {
		return listErr
	}
	if deleteErr != nil {
		return deleteErr
	}

	if len(opts.tracks) > 0 {
		tracks = filterTracks(tracks, opts.tracks)
	}

	maxVersionCode := findMaxVersionCode(tracks)
	if maxVersionCode < 0 {
		return fmt.Errorf("no version codes found")
	}

	fmt.Fprintf(stdout, "%d\n", maxVersionCode)
	return nil
}

func filterTracks(tracks []*androidpublisher.Track, allowed []string) []*androidpublisher.Track {
	set := make(map[string]bool, len(allowed))
	for _, t := range allowed {
		set[t] = true
	}
	var filtered []*androidpublisher.Track
	for _, t := range tracks {
		if set[t.Track] {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func findMaxVersionCode(tracks []*androidpublisher.Track) int64 {
	max := int64(-1)
	for _, track := range tracks {
		for _, release := range track.Releases {
			for _, vc := range release.VersionCodes {
				if vc > max {
					max = vc
				}
			}
		}
	}
	return max
}
