# Legacy Network Migrator

This repo contains a single script which facilitates migrating legacy networks to
VPC networks and performing necessary GKE control plane and node pool upgrades.

For more information on VPC and legacy networks, see
https://cloud.google.com/vpc/docs/vpc and https://cloud.google.com/vpc/docs/legacy,
respectively.

## Setup

The script uses Application Default Credentials for authentication.
If not already done, install the gcloud CLI from https://cloud.google.com/sdk/ and
run `gcloud auth application-default login`.

For more information, see
https://developers.google.com/identity/protocols/application-default-credentials.

## Execution

Warning: It is recommended that the migration tool be run within a maintenance window.
For more information, see
https://cloud.google.com/kubernetes-engine/docs/concepts/maintenance-windows-and-exclusions.

The script takes the name of a network as input and validates that
all underlying resources of the network can be upgraded. If validation fails for
any underlying resource, no resource - including the network - will be upgraded.

For the full list of script arguments and options, use the `--help` flag.

Please review any execution errors before re-attempting migration to ensure any
reported errors are transient.

The script is idempotent and forgoes any operation that is unnecessary to achieve
the desired state of the resources (i.e. network, control plane and node pools).
This allows the script to be executed without modifying arguments or flags to
re-attempt the migration if a resource upgrade were to fail.

For example, the script may be run against a network which has already been migrated.
The network upgrade will be skipped, and the script will proceed to upgrade the
cluster(s).

## Cluster upgrade options

Note: The script does not allow for downgrading the control plane nor node pool.

The options for the desired control plane and node pool versions accept version
aliases. See [cluster.UpdateMaster] and [nodePools.Update] for syntax and behavior.

[cluster.UpdateMaster]: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters/updateMaster
[nodePools.Update]: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters.nodePools/update

## Node pools

The time required to upgrade a Node pool is a function of the number of nodes and
may take a long time. An option is available to kick off the node pool upgrade and
continue the script execution.

As the script does not allow node pools to be downgraded, any node pool that is
running the latest node version will cause the validation to fail (node pools may
not be upgraded in-place).

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for details.

## License

Apache 2.0; see [`LICENSE`](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not supported by
Google and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
