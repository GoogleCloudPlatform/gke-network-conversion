# Legacy Network Conversion Tool for GKE

Legacy networks are deprecated and are no longer recommended. You can convert a
legacy network to a VPC network using the [Single-region conversion tool]. If your
legacy network contains Google Kubernetes Engine clusters, the cluster and node
pools must be upgraded. The upgrade ensures that all components operate correctly
after the conversion.

This repository contains a single script which converts legacy networks to VPC
networks and performs the necessary GKE control plane and node pool upgrades.

For more information on VPC and legacy networks, see [VPC network overview] and
[Legacy networks], respectively.

[VPC network overview]: https://cloud.google.com/vpc/docs/vpc
[Legacy networks]: https://cloud.google.com/vpc/docs/legacy

## Preparation

Read [Single-region conversion tool] to understand what changes are made in your
legacy network during conversion. Read [Converting a single region legacy network
to a VPC network] to understand the requirements and limitations of the single-region
conversion tool.

Read [Standard cluster upgrades] and [Manually upgrading a
cluster or node pool] to understand the process, limitations, and implications of performing cluster upgrades.

[Single-region conversion tool]: https://cloud.google.com/vpc/docs/legacy#single-region-conversion
[Converting a single region legacy network to a VPC network]: https://cloud.google.com/vpc/docs/using-legacy#convert
[Standard cluster upgrades]: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-upgrades
[Manually upgrading a cluster or node pool]: https://cloud.google.com/kubernetes-engine/docs/how-to/upgrading-a-cluster

## Setup

The script uses Application Default Credentials for authentication.
If not already done, install the gcloud CLI from <https://cloud.google.com/sdk/> and
run `gcloud auth application-default login`.

For more information, see [Authenticating as a service account].

To get started, clone this repo and build the binary:

```shell
git clone git@github.com:GoogleCloudPlatform/gke-network-conversionn.git && cd $(basename $_ .git)
make all
```

[Authenticating as a service account]: https://cloud.google.com/docs/authentication/production

## Execution

> **Note**: We recommend that you run the conversion tool within a maintenance window.
> For more information, see [Maintenance windows and exclusions].

The script takes the following input: the name of a network to convert and the control
plane and node versions that you want to upgrade to. The script validates that all underlying
resources of the network should and can be upgraded. If validation fails for any underlying
resource, no resource - including the network - is modified.

For the full list of script arguments and options, use the `--help` flag.

> **Best practice**: Before performing conversion, run the script in validation-only
> mode (`--validate-only=true`) and review the output.

Example script execution to convert the network and clusters:

```shell
gkeconvert                                       \
 --project=<PROJECT_ID>                          \
 --network=<NETWORK_NAME>                        \
 --control-plane-version=<CONTROL_PLANE_VERSION> \
 --node-version=<NODE_VERSION>                   \
 --validate-only=false
```

> **Note**: The default version alias for the node pool (`--node-version="-"`) will
> match the desired node pool version to the desired control plane version.

Review any execution errors before you attempt to convert the network again to ensure
that any reported errors are transient. For errors during the network conversion, see
[Troubleshooting a single-region conversion].

The script skips any operation that is unnecessary to ensure that the GKE resources (control
plane and node pools) are compatible with the VPC network. This allows the script to be
executed without modifying arguments or flags to reattempt the conversion if a resource
upgrade fails.

For example, the script can be run against a network which has already been converted.
The network upgrade is skipped, and the script proceeds to upgrade the cluster(s).

[Troubleshooting a single-region conversion]: https://cloud.google.com/vpc/docs/using-legacy#troubleshooting
[Maintenance windows and exclusions]: https://cloud.google.com/kubernetes-engine/docs/concepts/maintenance-windows-and-exclusions

## Cluster upgrade options

> **Note**: The script does not allow for downgrading a control plane or a node pool version.

The options for the desired control plane and node pool versions accept version
aliases. See [cluster.UpdateMaster] and [nodePools.Update] for syntax and behavior.

[cluster.UpdateMaster]: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters/updateMaster
[nodePools.Update]: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters.nodePools/update

## Node pools

The time required to upgrade a node pool is a function of the number of nodes and
the values for the [surge upgrade parameters], and [Pod disruption budgets]. Ensure these
values are tuned to balance application availability against node pool upgrade times. For
more information, see [Determining your optimal surge configuration].

Node pools cannot be upgraded in-place (desired version matches the current version).
Any node pool which requires an upgrade and is running the latest node version causes
the validation to fail because there is no version to upgrade the node pools to.

[surge upgrade parameters]: https://cloud.google.com/kubernetes-engine/docs/how-to/upgrading-a-cluster#surge
[Pod disruption budgets]: https://kubernetes.io/docs/tasks/run-application/configure-pdb/
[Determining your optimal surge configuration]: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-upgrades#optimizing-surge

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for details.

## License

Apache 2.0; see [`LICENSE`](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not supported by
Google and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
