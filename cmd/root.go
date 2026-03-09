package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	androidpublisher "google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"

	"github.com/YumNumm/google-play-cli/internal/auth"
	"github.com/YumNumm/google-play-cli/internal/publisher"
)

// ClientFactory creates a publisher.Client from credentials.
type ClientFactory func(ctx context.Context, credentials string) (publisher.Client, error)

// createPublisherService wraps androidpublisher.NewService for testability.
var createPublisherService = func(ctx context.Context, opts ...option.ClientOption) (*androidpublisher.Service, error) {
	return androidpublisher.NewService(ctx, opts...)
}

// DefaultClientFactory creates a real Google Play Publisher API client.
func DefaultClientFactory(ctx context.Context, credentials string) (publisher.Client, error) {
	resolver := auth.NewResolver()
	creds, err := resolver.Resolve(ctx, credentials)
	if err != nil {
		return nil, err
	}
	service, err := createPublisherService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher service: %w", err)
	}
	return publisher.NewClient(service), nil
}

func newRootCmd(factory ClientFactory) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "google-play-cli",
		Short:        "CLI tool for Google Play Developer API",
		Long:         "A CLI tool to interact with the Google Play Android Publisher API v3.\nSupports service account key and Workload Identity Federation (WIF) authentication.",
		SilenceUsage: true,
	}

	bundlesCmd := &cobra.Command{
		Use:   "bundles",
		Short: "Manage app bundles",
	}
	bundlesCmd.AddCommand(newBundlesPublishCmd(factory))
	rootCmd.AddCommand(bundlesCmd)
	rootCmd.AddCommand(newGetLatestBuildNumberCmd(factory))

	return rootCmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd(DefaultClientFactory).Execute()
}
