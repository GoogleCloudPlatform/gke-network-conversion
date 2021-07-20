/*
Copyright © 2021 Google

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"legacymigration/pkg"
	"legacymigration/pkg/clusters"
	"legacymigration/pkg/migrate"
	"legacymigration/pkg/networks"
	"legacymigration/pkg/operations"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
	computealpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
)

const (
	// Flag name constants.
	projectFlag                    = "project"
	containerBasePathFlag          = "container-base-url"
	networkFlag                    = "network"
	concurrentClustersFlag         = "concurrent-clusters"
	desiredControlPlaneVersionFlag = "control-plane-version"
	desiredNodeVersionFlag         = "node-version"
	pollingIntervalFlag            = "polling-interval"
	pollingDeadlineFlag            = "polling-deadline"
	inPlaceControlPlaneUpgradeFlag = "in-place-control-plane"
	validateOnlyFlag               = "validate-only"

	ConcurrentNetworks  = 1
	ConcurrentNodePools = 1
)

type migrateOptions struct {
	// Options set by flags.
	projectID                  string
	containerBasePath          string
	selectedNetwork            string
	concurrentClusters         uint16
	desiredControlPlaneVersion string
	desiredNodeVersion         string
	inPlaceControlPlaneUpgrade bool
	validateOnly               bool
	pollingInterval            time.Duration
	pollingDeadline            time.Duration

	// Field used for faking clients during tests.
	fetchClientFunc func(ctx context.Context, basePath string, authedClient *http.Client) (*pkg.Clients, error)

	// Options set during Complete
	clients   *pkg.Clients
	migrators []migrate.Migrator
}

var (
	rootCmd = newRootCmd()
)

// rootCmd represents the base command when called without any subcommands
func newRootCmd() *cobra.Command {
	o := migrateOptions{
		fetchClientFunc: fetchClients,
	}
	ctx, cancel := context.WithCancel(context.Background())

	cmd := &cobra.Command{
		Use:   "gkeconvert",
		Short: "Convert a GCE legacy network to a VPC network and upgrade GKE clusters.",
		Long: `This script converts a GCE legacy network to a VPC network (by switching
the network to custom subnet mode). It then performs GKE cluster upgrades to ensure
the clusters are compatible with a VPC network.`,

		PreRun: func(cmd *cobra.Command, args []string) {
			cobra.CheckErr(o.ValidateFlags())
			setupCloseHandler(cancel)
		},
		Run: func(cmd *cobra.Command, args []string) {
			cobra.CheckErr(o.Complete(ctx))
			cobra.CheckErr(o.Run(ctx))
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&o.projectID, projectFlag, "p", o.projectID, "project ID")

	// Target network options.
	flags.StringVarP(&o.selectedNetwork, networkFlag, "n", o.selectedNetwork, "GCE network to process.")

	// Concurrency options.
	flags.Uint16VarP(&o.concurrentClusters, concurrentClustersFlag, "C", 1, "Number of clusters per network to upgrade concurrently.")

	// Polling options.
	flags.DurationVar(&o.pollingInterval, pollingIntervalFlag, 15*time.Second, "Period between polling attempts.")
	flags.DurationVar(&o.pollingDeadline, pollingDeadlineFlag, 24*time.Hour, "Deadline for a long running operation to complete (e.g. to upgrade a cluster node pool).")

	// Cluster upgrade options.
	flags.StringVar(&o.desiredControlPlaneVersion, desiredControlPlaneVersionFlag, o.desiredControlPlaneVersion,
		`Desired GKE version for all cluster control planes.
For more information, see https://cloud.google.com/kubernetes-engine/versioning#versioning_scheme
Note:
  This version must be equal to or greater than the lowest control plane version on the network.`)
	flags.StringVar(&o.desiredNodeVersion, desiredNodeVersionFlag, o.desiredNodeVersion,
		`Desired GKE version for all cluster nodes. For more information, see https://cloud.google.com/kubernetes-engine/versioning#versioning_scheme
Note:
  This version must be greater than the lowest node pool version on the network as node pools cannot be upgraded in-place.`)

	flags.BoolVar(&o.inPlaceControlPlaneUpgrade, inPlaceControlPlaneUpgradeFlag, false,
		`Perform in-place control plane upgrade for all clusters.`)

	flags.BoolVar(&o.validateOnly, validateOnlyFlag, true,
		`Only run validation on the network and cluster resources; do not perform conversion`)

	// Test options.
	flags.StringVar(&o.containerBasePath, containerBasePathFlag, o.containerBasePath, "Custom URL for the container API endpoint (for testing).")

	cmd.MarkFlagRequired(projectFlag)
	cmd.MarkFlagRequired(networkFlag)
	cmd.MarkFlagRequired(desiredNodeVersionFlag)
	flags.MarkHidden(containerBasePathFlag)

	return cmd
}

// Execute runs the root command.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

// ValidateFlags ensures flags values are valid for execution.
func (o *migrateOptions) ValidateFlags() error {
	if o.projectID == "" {
		return fmt.Errorf("--%s not provided or empty", projectFlag)
	}
	if o.selectedNetwork == "" {
		return fmt.Errorf("--%s not provided or empty", networkFlag)
	}

	// Concurrency validation.
	if o.concurrentClusters < 1 {
		return fmt.Errorf("--%s must be an integer greater than 0", concurrentClustersFlag)
	}

	// Polling validation.
	if o.pollingInterval < 10*time.Second {
		return fmt.Errorf("--%s must greater than or equal to 10 seconds. Note: Upgrade operations times are O(minutes)", pollingIntervalFlag)
	}
	if o.pollingDeadline < 5*time.Minute {
		return fmt.Errorf("--%s must greater than or equal to 5 minutes. Note: Upgrade operations times are O(minutes)", pollingDeadlineFlag)
	}
	if o.pollingInterval > o.pollingDeadline {
		return fmt.Errorf("--%s=%v must be greater than --%s=%v", pollingDeadlineFlag, o.pollingDeadline, pollingIntervalFlag, o.pollingInterval)
	}

	// Version validation.
	if (o.desiredControlPlaneVersion == "" && !o.inPlaceControlPlaneUpgrade) ||
		(o.desiredControlPlaneVersion != "" && o.inPlaceControlPlaneUpgrade) {
		return fmt.Errorf("specify --%s or provide a version for --%s, but not both", inPlaceControlPlaneUpgradeFlag, desiredControlPlaneVersionFlag)
	}
	if o.desiredControlPlaneVersion != "" {
		if err := clusters.IsFormatValid(o.desiredControlPlaneVersion); err != nil {
			return fmt.Errorf("--%s=%q is not valid: %v", desiredControlPlaneVersionFlag, o.desiredControlPlaneVersion, err)
		}
	}
	if err := clusters.IsFormatValid(o.desiredNodeVersion); err != nil {
		return fmt.Errorf("--%s=%q is not valid: %v", desiredNodeVersionFlag, o.desiredNodeVersion, err)
	}
	// Use of `-` or `latest` aliases are validated later at the control plane and node pool level.
	if !o.inPlaceControlPlaneUpgrade &&
		o.desiredControlPlaneVersion != clusters.DefaultVersion &&
		o.desiredControlPlaneVersion != clusters.LatestVersion &&
		o.desiredNodeVersion != clusters.DefaultVersion &&
		o.desiredNodeVersion != clusters.LatestVersion {
		if err := clusters.IsWithinVersionSkew(o.desiredNodeVersion, o.desiredControlPlaneVersion, clusters.MaxVersionSkew); err != nil {
			return err
		}
	}

	return nil
}

// Complete cascades down the resource hierarchy, ensuring that all descendent migrators are initialized.
func (o *migrateOptions) Complete(ctx context.Context) error {
	authedClient, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	o.clients, err = o.fetchClientFunc(ctx, o.containerBasePath, authedClient)
	if err != nil {
		return err
	}

	handler := operations.NewHandler(o.pollingInterval, o.pollingDeadline)
	options := &clusters.Options{
		ConcurrentNodePools:        ConcurrentNodePools,
		DesiredControlPlaneVersion: o.desiredControlPlaneVersion,
		DesiredNodeVersion:         o.desiredNodeVersion,
		InPlaceControlPlaneUpgrade: o.inPlaceControlPlaneUpgrade,
	}

	factory := func(n *compute.Network) migrate.Migrator {
		return networks.New(o.projectID, n, handler, o.clients, o.concurrentClusters, options)
	}

	log.Infof("Fetching network %s for project %q", o.selectedNetwork, o.projectID)

	ns, err := o.clients.Compute.ListNetworks(ctx, o.projectID)
	if err != nil {
		return fmt.Errorf("error listing networks: %w", err)
	}

	o.migrators = make([]migrate.Migrator, 0)
	for _, n := range ns {
		if n.Name == o.selectedNetwork {
			o.migrators = append(o.migrators, factory(n))
		}
	}

	if len(o.migrators) == 0 {
		return fmt.Errorf("unable to find network %s", o.selectedNetwork)
	}

	return nil
}

// Run cascades down the resource hierarchy, initializing, validating, and converting all descendent migrators.
func (o *migrateOptions) Run(ctx context.Context) error {
	sem := make(chan struct{}, ConcurrentNetworks)

	log.Info("Initialize objects for conversion.")
	if err := migrate.Complete(ctx, sem, o.migrators...); err != nil {
		return err
	}

	log.Info("Validate resources for conversion.")
	if err := migrate.Validate(ctx, sem, o.migrators...); err != nil {
		return err
	}

	if o.validateOnly {
		log.Infof("--%s==true; skipping conversion.", validateOnlyFlag)
		return nil
	}

	log.Info("Initiate resource conversion.")
	return migrate.Migrate(ctx, sem, o.migrators...)
}

// setupCloseHandler cancels the shared context when the user hits ctrl-c.
func setupCloseHandler(cancel context.CancelFunc) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()
}

func getRetryableClientOption(retry int, waitMin, waitMax time.Duration, authedClient *http.Client) option.ClientOption {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = retry
	retryClient.RetryWaitMin = waitMin
	retryClient.RetryWaitMax = waitMax
	retryClient.Logger = nil
	retryClient.CheckRetry = retryPolicy()

	c := retryClient.StandardClient()
	c.Transport.(*retryablehttp.RoundTripper).Client.HTTPClient = authedClient
	return option.WithHTTPClient(c)
}

func fetchClients(ctx context.Context, basePath string, authedClient *http.Client) (*pkg.Clients, error) {
	opt := getRetryableClientOption(3, 5*time.Second, 30*time.Second, authedClient)
	computeService, err := compute.NewService(ctx, opt)
	if err != nil {
		return nil, err
	}
	containerService, err := container.NewService(ctx, opt)
	if err != nil {
		return nil, err
	}

	// Retry for up-to 5 minutes for Compute Alpha API calls.
	alphaOpt := getRetryableClientOption(5, 5*time.Second, 160*time.Second, authedClient)
	computeServiceAlpha, err := computealpha.NewService(ctx, alphaOpt)
	if err != nil {
		return nil, err
	}

	if basePath != "" {
		containerService.BasePath = basePath
	}
	return &pkg.Clients{
		Compute: &pkg.Compute{
			V1:    computeService,
			Alpha: computeServiceAlpha,
		},
		Container: &pkg.Container{V1: containerService},
	}, nil
}

func retryPolicy() func(ctx context.Context, resp *http.Response, err error) (bool, error) {
	return func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		shouldRetry, newErr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		if newErr != nil || shouldRetry == true {
			return shouldRetry, newErr
		}

		// GCE returns 403 as a RateLimiting response code.
		if resp.StatusCode == 403 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return true, err
			}
			return true, errors.New(string(body))
		}
		return false, nil
	}
}
