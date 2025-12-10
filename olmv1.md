# Quick Reference
  - [create catalog](#olmv1-create-catalog) - Create a new ClusterCatalog
  - [install extension](#olmv1-install-extension) - Install a ClusterExtension
  - [delete catalog](#olmv1-delete-catalog) - Delete ClusterCatalog(s)
  - [delete extension](#olmv1-delete-extension) - Delete ClusterExtension(s)
  - [update catalog](#olmv1-update-catalog) - Update ClusterCatalog fields
  - [update extension](#olmv1-update-extension) - Update ClusterExtension fields
  - [get catalog](#olmv1-get-catalog) - List/view ClusterCatalogs
  - [get extension](#olmv1-get-extension) - List/view ClusterExtensions
  - [search catalog](#olmv1-search-catalog) - Search for packages
  - [workflows](#recommended-workflows) - Recommended workflows for common tasks

# Kubectl Operator OLMv1 Commands

Most of the kubectl operator plugin subcommands are aimed toward managing OLMv0 operators.

[OLMv1](https://github.com/operator-framework/operator-controller/) has two main on-cluster CRDs, both of which can be managed by the kubectl operator `olmv1` subcommand. These are:
- [`ClusterCatalogs`](https://github.com/operator-framework/operator-controller/blob/main/api/v1/clustercatalog_types.go): A CR used to provide a curated collection of packages for users to install and upgrade from, backed by an image reference to a File Based Catalog (FBC) 
- [`ClusterExtensions`](https://github.com/operator-framework/operator-controller/blob/main/api/v1/clusterextension_types.go): A CR expressing the current status and desired state of a package on cluster. It must provide the package name to be installed from available `ClusterCatalogs`, but may also specify restrictions on install and upgrade discovery.

Within the repository, these are defined in `internal/cmd/olmv1.go`, which in turn references `internal/cmd/internal/olmv1` and `internal/pkg/v1`.

```bash
$ kubectl operator olmv1 --help

Manage OLMv1 resources like clusterextensions and clustercatalogs from the command line.

Usage:
  operator olmv1 [command]

Available Commands:
  create      Create a resource
  delete      Delete a resource
  get         Display one or many resource(s)
  install     Install a resource
  search      Search for packages
  update      Update a resource
```

Of the global flags, only `--help` is relevant to `olmv1` and its subcommands. The `olmv1` subcommands are detailed as follows.

<br/>
<br/>

---

## `olmv1 create`
Create `olmv1` resources, currently supports only `ClusterCatalogs`.

```bash
Create a resource

Usage:
  operator olmv1 create [command]

Available Commands:
  catalog     Create a new catalog
```
<br/>

### `olmv1 create catalog`
Create a new `ClusterCatalog` with the provided name and catalog image reference.

```bash
Create a new catalog

Usage:
  operator olmv1 create catalog <catalog_name> <image_source_ref> [flags]

Aliases:
  catalog, catalogs <catalog_name> <image_source_ref>

Flags:
      --available string                   determines whether a catalog should be active and serving data. Setting the flag to false means the catalog will not serve its contents.
      --cleanup-timeout duration           the amount of time to wait before cancelling cleanup after a failed creation attempt. (default 1m0s)
      --dry-run string                     display the object that would be sent on a request without applying it. One of: (All)
      --labels stringToString              labels to add to the catalog. Set a label's value as empty to remove it. (default [])
  -o, --output string                      output format for dry-run manifests. One of: (json, yaml)
      --priority int32                     relative priority of the catalog among all on-cluster catalogs for installing or updating packages. A higher number equals greater priority; negative values indicate less priority than the default.
      --source-poll-interval-minutes int   the interval in minutes to poll the catalog's source image for new content. Only valid for tag based source image references. Set to 0 or -1 to disable polling.
```

The flags allow for setting most mutable fields:
- `--available`: Sets whether the `ClusterCatalog` should be actively serving and make its contents available on cluster. Default: true.
  Valid values:
  - `--available=true` or `--available` (flag alone): Catalog will serve content
  - `--available=false`: Catalog created but won't serve (useful for pre-staging)
  Note: The flag defaults to true, so omitting it creates a serving catalog.

- `--cleanup-timeout`: If a `ClusterCatalog` creation attempt fails due to the resource never becoming healthy, `olmv1` cleans up by deleting the failed resource, with a timeout specified by `--cleanup-timeout`. Default: 1 minute (1m)
- `--labels`: Additional labels to add to the newly created `ClusterCatalog` as `key=value` pairs. This flag may be specified multiple times.
- `--priority`: Integer priority used for ordering `ClusterCatalogs` in case two extension packages have the same name across `ClusterCatalogs`, with a higher value indicating greater relative priority. Default: 0
- `--source-poll-interval-minutes`: The polling interval to check for changes if the `ClusterCatalog` source image provided is not a digest based image, i.e, if it is referenced by tag. Set to 0 to disable polling. Default: 10
- `--dry-run`: Generate the manifest that would be applied with the command without actually applying it to the cluster.
- `--output`: The format for displaying manifests if `--dry-run` is specified.

The command requires at minimum a resource name and image reference.

Minimal command example:
```bash
$ kubectl operator olmv1 create catalog mycatalog myorg/mycatalogrepo:tag
```

A full example with optional flags may look like:
```bash
$ kubectl operator olmv1 create catalog mycatalog myorg/mycatalogrepo:tag --labels foo=bar --available=false --priority 100 --source-poll-interval-minutes 5 --cleanup-timeout 5m
catalog "mycatalog" created
```
The example creates a catalog without enabling it on the cluster. The successful creation can be verified as follows:
```bash
$ kubectl operator olmv1 get catalog mycatalog
NAME       AVAILABILITY  PRIORITY  LASTUNPACKED  SERVING  AGE
mycatalog  Unavailable   100                     False    19s
```
The catalog is present but Unavailable and not serving, as expected from the `--available=false` flag passed to the example.

With `--dry-run`, the ClusterCatalog will be displayed but not applied. Without an accompanying `--output` flag, `--dry-run` only verifies whether the apply may succeed.

A failed dry-run:
```bash
$ kubectl operator olmv1 create catalog mycatalog myorg/mycatalogrepo:tag --dry-run=All
failed to create catalog "mycatalog": clustercatalogs.olm.operatorframework.io "mycatalog" already exists
```

A successful dry-run:
```bash
$ kubectl operator olmv1 create catalog mycatalog1 myorg/mycatalogrepo:tag --dry-run=All
catalog "mycatalog1" created (dry run)
```

When passing the `--output` flag to a `--dry-run` command, the manifest to be applied is returned, which allows the object to be verified, or even piped through tools like `yq` or `jq` to make modifications to the manifest before applying it to the cluster.

```bash
$ kubectl operator olmv1 create catalog mycatalog1 myorg/mycatalogrepo:tag --dry-run=All -o yaml
apiVersion: olm.operatorframework.io/v1
kind: ClusterCatalog
metadata:
  creationTimestamp: "2025-11-14T12:48:43Z"
  generation: 1
  labels:
    olm.operatorframework.io/metadata.name: mycatalog1
  managedFields:
  - apiVersion: olm.operatorframework.io/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        .: {}
        f:availabilityMode: {}
        f:priority: {}
        f:source:
          .: {}
          f:image:
            .: {}
            f:pollIntervalMinutes: {}
            f:ref: {}
          f:type: {}
    manager: kubectl-operator
    operation: Update
    time: "2025-11-14T12:48:43Z"
  name: mycatalog1
  uid: 31278405-b6d0-4555-b9b1-15aa23c649c8
spec:
  availabilityMode: Available
  priority: 0
  source:
    image:
      pollIntervalMinutes: 10
      ref: myorg/mycatalogrepo:tag
    type: Image
status: {}
```

```bash
$ kubectl operator olmv1 create catalog mycatalog1 myorg/mycatalogrepo:tag --dry-run=All -o json
{
    "kind": "ClusterCatalog",
    "apiVersion": "olm.operatorframework.io/v1",
    "metadata": {
        "name": "mycatalog1",
        "uid": "89919d7e-226f-43e2-88f9-91fff3936823",
        "generation": 1,
        "creationTimestamp": "2025-11-14T12:48:46Z",
        "labels": {
            "olm.operatorframework.io/metadata.name": "mycatalog1"
        },
        "managedFields": [
            {
                "manager": "kubectl-operator",
                "operation": "Update",
                "apiVersion": "olm.operatorframework.io/v1",
                "time": "2025-11-14T12:48:46Z",
                "fieldsType": "FieldsV1",
                "fieldsV1": {
                    "f:spec": {
                        ".": {},
                        "f:availabilityMode": {},
                        "f:priority": {},
                        "f:source": {
                            ".": {},
                            "f:image": {
                                ".": {},
                                "f:pollIntervalMinutes": {},
                                "f:ref": {}
                            },
                            "f:type": {}
                        }
                    }
                }
            }
        ]
    },
    "spec": {
        "source": {
            "type": "Image",
            "image": {
                "ref": "myorg/mycatalogrepo:tag",
                "pollIntervalMinutes": 10
            }
        },
        "priority": 0,
        "availabilityMode": "Available"
    },
    "status": {}
}
```

Prerequisites
Before creating a catalog, ensure:
1. Catalog image is accessible
  - Image must be pullable from the cluster
  - Verify: `docker pull <image>` or `podman pull <image>` works
2. Catalog image contains valid File-Based Catalog (FBC)
  - Image must contain a valid FBC structure
  - Invalid catalogs will be created but never become healthy/serving
  - Verify: if `opm` is installed, the catalog may be verified by `opm render <image> >/dev/null` 

Common errors:
- A ClusterCatalog with the given name already exists
```bash
$ kubectl operator olmv1 create catalog mycatalog myorg/mycatalogrepo:tag 
failed to create catalog "mycatalog": clustercatalogs.olm.operatorframework.io "mycatalog" already exists
```
  Fix: If the existing catalog is meant to be updated, use the `kubectl operator olmv1 update catalog` command instead. If the catalog is not meant to replace the existing one, consider using a different catalog name.
- ClusterCatalog is set to serve(i.e, Available=true) but never becomes healthy
  Possible causes: the provided catalog image does not exist, is unreachable or does not contain a catalog. Verify whether the image is reachable and valid. More information about the error can be acquired from the status conditions.
```bash
$ kubectl operator olmv1 get catalog mycatalog1 -o yaml | yq eval '.status.conditions[] | select(.type=="Progressing")'
lastTransitionTime: "2025-11-14T14:17:44Z"
message: 'source catalog content: terminal error: error parsing image reference "myorg/mycatalogrepo:tag": repository name must be canonical'
observedGeneration: 1
reason: Blocked
status: "False"
type: Progressing

```
<br/>
<br/>

---
## olmv1 install
Install resources with olmv1, currently supports `ClusterExtensions`.

```bash
Install a resource

Usage:
  operator olmv1 install [command]

Available Commands:
  extension   Install an extension
```
<br/>

### olmv1 install extension

Install a new `ClusterExtension` with the provided name.

```bash
Install an extension

Usage:
  operator olmv1 install extension <extension_name> [flags]

Flags:
  -l, --catalog-selector string                 selector (label query) to filter catalogs to search for the package, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.
  -c, --channels strings                        channels to be used for getting updates. If omitted, extension versions in all channels will be considered for upgrades. When used with '--version', only package versions meeting both constraints will be considered.
      --cleanup-timeout duration                the amount of time to wait before cancelling cleanup after a failed creation attempt (default 1m0s)
      --crd-upgrade-safety-enforcement string   policy for preflight CRD Upgrade safety checks. One of: [Strict None], (default Strict)
      --dry-run string                          display the object that would be sent on a request without applying it. One of: (All)
      --labels stringToString                   labels to add to the extension. Set a label's value as empty to remove that label (default [])
  -n, --namespace string                        namespace to install the operator in (default "olmv1-system")
  -o, --output string                           output format for dry-run manifests. One of: (json, yaml)
  -p, --package-name string                     package name of the operator to install. Required.
  -s, --service-account string                  service account name to use for the extension installation (default "default")
      --upgrade-constraint-policy string        controls whether the package upgrade path(s) defined in the catalog are enforced. One of [CatalogProvided SelfCertified], (default CatalogProvided)
  -v, --version string                          version (or version range) in semver format to limit the allowable package versions to. If used with '--channel', only package versions meeting both constraints will be considered.
```

The flags allow for setting most mutable fields:
- **`-n`, `--namespace`**: The namespace to install any namespace-scoped resources created by the `ClusterExtension`. <span style="color:red">**Required**</span>
- **`-p`, `--package-name`**: Name of the package to install. <span style="color:red">**Required**</span>
- `-c`, `--channels`: An optional list of channels within a package to restrict searches for an installable version to. 
- `-d`, `--cleanup-timeout`: If a `ClusterExtension` creation attempt fails due to the resource never becoming healthy, `olmv1` cleans up by deleting the failed resource, with a timeout specified by `--cleanup-timeout`. Default: 1 minute (1m)
- `-s`, `--service-account`: Name of the ServiceAccount present in the namespace specified by `--namespace` to use for creating and managing resources for the new `ClusterExtension`. If not specified, the command expects a ServiceAccount `default` to be present in the namespace provided by `--namespace` with the required permissions to create and manage all the resources the `ClusterExtension` may require.
- `-v`, `--version`: A version or version range to restrict search for an installable version to.  If specified along with `--channels`, only versions in the version range belonging to one or more of the channels specified will be allowed.
  Valid version range format examples:
  - Exact: `--version 1.2.3`
  - Range: `--version ">=1.0.0 <2.0.0"`
  - Wildcard: `--version 1.2.x` (any patch version of 1.2)
  - Minimum: `--version ">=1.5.0"`
    Example combinations:
```bash
# Latest from stable channel
--channels stable

# Version 1.2.x from any channel
--version 1.2.x

# Version 1.2.x from stable OR beta channels
--channels stable,beta --version 1.2.x
```
- `--dry-run`: Generate the manifest that would be applied with the command without actually applying it to the cluster.
- `--output`: The format for displaying manifests if `--dry-run` is specified.
- `--catalog-selector`: Limit the sources that the package specified by the ClusterExtension can be installed from to ClusterCatalogs matching the provided label selector. Only useful if the ClusterCatalogs on cluster have been labelled meaningfully, such as by maturity, provider etc. eg:
```bash
--catalog-selector foo=bar
# Multiple label constraints
--catalog-selector foo=bar,bar=baz
```
- `--labels`: Labels to be added to the new ClusterExtension.
- `--upgrade-constraint-policy`:Controls how upgrade versions are picked for the ClusterExtension. If set to `SelfCertified`, upgrade to any version of the package (even earlier versions, i.e, downgrades) are allowed. If set to `CatalogProvided`, only upgrade paths mentioned in the ClusterCatalog are allowed for the package. Note that `SelfCertified` upgrades may be unsafe and lead to data loss.
- `--crd-upgrade-safety-enforcement`: configures pre-flight CRD Upgrade safety checks. If set to `Strict`, an upgrade will be blocked if it has breaking changes to a CRD on cluster. If set to `None`, this pre-flight check is skipped, which may cause unsafe changes during installs and upgrades.
<br/>
<br/>

The command requires at minimum a resource name, `--namespace` and `--package-name`.
A minimal example:
```bash
$ kubectl operator olmv1 install extension myex --namespace myns --package-name prometheus-operator
```

A more complete example may look like:
```bash
$ kubectl operator olmv1 install extension testoperator --package-name prometheus --service-account olmv1-operators --namespace olmv1-system
extension "testoperator" created
```

This creates an extension for the `prometheus` package that installs the latest available version of that package. Success may be verified with:
```bash
$ kubectl operator olmv1 get extension
NAME          INSTALLED BUNDLE            VERSION  SOURCE TYPE  INSTALLED  PROGRESSING  AGE
testoperator  prometheusoperator.v0.70.0  0.70.0   Catalog      True       True         21s
```
A successful installation must eventually have `INSTALLED` be `True`. Poll until this condition is met or till the manifest shows error messages in its conditions.

A failed install will have an error message on the `Progressing` condition on its `status.conditions` field in a detailed view of the resource through `kubectl operator olmv1 get extension <extension name> -o yaml`
```bash
$ kubectl operator olmv1 get extension -o yaml testoperators | yq eval '.status.conditions.[] | select(.type=="Installed")'
lastTransitionTime: "2025-11-14T13:28:25Z"
message: 'service account "olmv1-operators" not found in namespace "olmv1-system": unable to authenticate with the Kubernetes cluster.'
observedGeneration: 1
reason: Failed
status: Unknown
type: Installed
```

Prerequisites:
  Before installing an extension, ensure:
  1. **At least one ClusterCatalog exists and is serving**
     - Verify: `kubectl operator olmv1 get catalog` shows at least one catalog with `SERVING: True`
  2. **Target namespace exists**
     - Verify: `kubectl get namespace <namespace-name>`
     - Create if needed: `kubectl create namespace <namespace-name>`
  3. **ServiceAccount exists with required permissions** (if not using default)
     - Verify: `kubectl get serviceaccount <sa-name> -n <namespace>`
  3. **ClusterExtension with the desired name does not already exist**
     - Verify: `kubectl operator olmv1 get extension <extension-name>`
     - If it already exists, use a different name or update the existing ClusterExtension with `kubectl operator olmv1 update extension`

Common errors:
- A ClusterExtension with the specified name already exists on the cluster
```bash
$ kubectl operator olmv1 install extension myex --namespace myns --package-name prometheus-operator
failed to install extension "myex": clusterextensions.olm.operatorframework.io "myex" already exists
```
  Fix: If the extension is meant to be updated, use the `kubectl olmv1 update extension` command instead. If the ClusterExtension is not meant to replace the original ClusterExtension, use a different name for the new ClusterExtension.
- The package specified cannot be found
```bash
$ kubectl operator olmv1 install extension testoperator --package-name prometheusoperator 
failed to install extension "testoperator": cluster extension "testoperator" did not finish installing: no bundles found for package "prometheusoperator"
```
  Cause:
  - The catalog providing the package may not be serving or may be unhealthy.
    Verify with `kubectl olmv1 get catalog -o yaml` that the required catalogs have healthy statuses and have `spec.availabilityMode` set to `Available`.
  - The package name may be incorrect.
    Verify with `kubectl olmv1 search catalog` to see if the package name is spelled correctly.
  - The `--version` or `--channel` constraints may be too restrictive if present.
    Verify with `kubectl olmv1 search catalog --package=<package_name>` with and without `--list-versions` to see if either argument is incorrect.
- The ServiceAccount to use is missing
```bash
$ kubectl operator olmv1 install extension testoperator --package-name prometheus --service-account olmv1-operators --namespace olmv1-system
failed to install extension prometheus: cluster extension "testoperator" did not finish installing: installation cannot proceed due to missing ServiceAccount
```
  Fix: Create the ServiceAccount with the required permissions, or use an appropriate existing ServiceAccount. Note that this error may also occur if the namespace is incorrect, since the command only checks the provided namespace for the ServiceAccount.
- ServiceAccount provided is missing the expected permissions to complete the installation.
```bash
$ kubectl operator olmv1 install extension testoperator --package-name prometheus --namespace=olmv1-system
failed to install extension prometheus: cluster extension "testoperator" did not finish installing: error for resolved bundle "prometheusoperator.v0.70.0" with version "0.70.0": failed to get release state using server-side dry-run: Unable to continue with install: could not get information about the resource ServiceAccount "prometheus-operator" in namespace "olmv1-system": serviceaccounts "prometheus-operator" is forbidden: User "system:serviceaccount:olmv1-system:default" cannot get resource "serviceaccounts" in API group "" in the namespace "olmv1-system"
```
  Fix: Add the required permissions to the ServiceAccount through adding or updating the right RBAC resources.
- provided namespace does not exist
```bash
$ kubectl operator olmv1 install extension testoperators --package-name prometheus --service-account olmv1-operators --namespace randomns
failed to install extension prometheus: cluster extension "testoperators" did not finish installing: installation cannot proceed due to missing ServiceAccount
```
  Fix: Ensure the namespace exists. If it doesn't, either create it or use an existing namespace.
```bash
$ kubectl get ns randomns
Error from server (NotFound): namespaces "randomns" not found
```

<br/>
<br/>

---

## olmv1 delete

Delete `olmv1` resources.

```bash
Delete a resource

Usage:
  operator olmv1 delete [command]

Available Commands:
  catalog     Delete either a single or all of the existing catalogs
  extension   Delete an extension
```
<br/>

### olmv1 delete catalog

Delete a `ClusterCatalog` by name. Specifying the `--all` flag deletes all existing `ClusterCatalogs`, and cannot be used when a resource name is passed as an argument.


```bash
Delete either a single or all of the existing catalogs

Usage:
  operator olmv1 delete catalog [catalog_name] [flags]

Aliases:
  catalog, catalogs [catalog_name]

Flags:
  -a, --all              delete all catalogs
      --dry-run string   display the object that would be sent on a request without applying it. One of: (All)
  -o, --output string    output format for dry-run manifests. One of: (json, yaml)
```

The command requires exactly one of a resource name or the `--all` flag:
```bash
$ kubectl operator olmv1 delete catalog mycatalog
$ kubectl operator olmv1 delete catalog --all
```

Using `--all` along with a resource name is invalid. For example, the following command is invalid:
```bash
$ kubectl operator olmv1 delete catalog mycatalog --all

failed to delete catalog: cannot specify both --all and a catalog name
```

<br/>

### olmv1 delete extension

Delete a `ClusterExtension` by name. Specifying the `--all` flag deletes all existing `ClusterExtensions`, and cannot be used when a resource name is passed as an argument.

```bash
Usage:
  operator olmv1 delete extension [extension_name] [flags]

Aliases:
  extension, extensions [extension_name]

Flags:
  -a, --all              delete all extensions
      --dry-run string   display the object that would be sent on a request without applying it. One of: (All)
  -o, --output string    output format for dry-run manifests. One of: (json, yaml)
```

The command requires exactly one of a resource name or the `--all` flag:
```bash
$ kubectl operator olmv1 delete extension myex
$ kubectl operator olmv1 delete extension --all
```

Using `--all` along with a resource name is invalid. For example, the following command is invalid:
```bash
$ kubectl operator olmv1 delete extension myex --all

failed to delete extension: cannot specify both --all and an extension name
```

<br/>
<br/>

---

## olmv1 update

Update mutable fields on `olmv1` resources.

```bash
Update a resource

Usage:
  operator olmv1 update [command]

Available Commands:
  catalog     Update a catalog
  extension   Update an extension
```
<br/>

### olmv1 update catalog

Update supported mutable fields on a `ClusterCatalog` specified by name.

```bash
Update a catalog

Usage:
  operator olmv1 update catalog <catalog_name> [flags]

Flags:
      --available string                   determines whether a catalog should be active and serving data. Setting the flag to false means the catalog will not serve its contents. Set to true by default for new catalogs.
      --dry-run string                     display the object that would be sent on a request without applying it. One of: (All)
      --ignore-unset                       set to false to revert all values not specifically set with flags in the command to their default as defined by the clustercatalog customresoucedefinition. (default true)
      --image string                       image reference for the catalog source. Leave unset to retain the current image.
      --labels stringToString              labels to add to the catalog. Set a label's value as empty to remove it. (default [])
  -o, --output string                      output format for dry-run manifests. One of: (json, yaml)
      --priority int32                     relative priority of the catalog among all on-cluster catalogs for installing or updating packages. A higher number equals greater priority; negative values indicate less priority than the default.
      --source-poll-interval-minutes int   the interval in minutes to poll the catalog's source image for new content. Only valid for tag based source image references. Set to 0 or -1 to disable polling.

```

The flags allow for setting most mutable fields:
- `--ignore-unset`: Sets the behavior of unspecified or empty flags, whether they should be ignored, preserving the current value on the resource, or treated as valid and used to set the field values to their default value.
- `--available`: Sets whether the `ClusterCatalog` should be actively serving and making its contents available on cluster. Valid values: `true`|`false`.
- `--image`: Update the image reference for the `ClusterCatalog`.
- `--labels`: Additional labels to add to the `ClusterCatalog` as `key=value` pairs. This flag may be specified multiple times. Setting the value of a label to an empty string deletes the label from the resource.
- `--priority`: Integer priority used for ordering `ClusterCatalogs` in case two extension packages have the same name across `ClusterCatalogs`, with a higher value indicating greater relative priority.
- `--source-poll-interval-minutes`: The polling interval to check for changes if the `ClusterCatalog` source image provided is not a digest based image, i.e, if it is referenced by tag. Set to 0 or -1 to disable polling.
- `--dry-run`: Generate the manifest that would be applied with the command without actually applying it to the cluster.
- `--output`: The format for displaying manifests if `--dry-run` is specified.
<br/>

To update specific fields on a catalog, like adding a new label or setting availability, the required flag may be used on its own:
```bash
$ kubectl operator olmv1 update catalog mycatalog --labels newlabel=newkey --labels labeltoremove=
$ kubectl operator olmv1 update catalog --available true
```

To reset a specific field on a catalog to its default, the value needs to be provided or all existing fields must be specified with `--ignore-unset`.
```bash
$ kubectl operator olmv1 update catalog mycatalog --available true
$ kubectl operator olmv1 update catalog mycatalog --ignore-unset=false --priority=10 --source-poll-interval-minutes=-1 --image=myorg/mycatalogrepo:tag --labels existing1=labelvalue1 --labels existing2=labelvalue2
```

### olmv1 update extension

Update supported mutable fields on a `ClusterExtension` specified by name.

```bash
Update an extension

Usage:
  operator olmv1 update extension <extension> [flags]

Flags:
  -l, --catalog-selector string                 selector (label query) to filter catalogs to search for the package, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.
  -c, --channels strings                        channels to be used for getting updates. If omitted, extension versions in all channels will be considered for upgrades. When used with '--version', only package versions meeting both constraints will be considered.
      --crd-upgrade-safety-enforcement string   policy for preflight CRD Upgrade safety checks. One of: [Strict None], (default Strict)
      --dry-run string                          display the object that would be sent on a request without applying it. One of: (All)
      --ignore-unset                            set to false to revert all values not specifically set with flags in the command to their default as defined by the clusterextension customresoucedefinition. (default true)
      --labels stringToString                   labels to add to the extension. Set a label's value as empty to remove that label (default [])
  -o, --output string                           output format for dry-run manifests. One of: (json, yaml)
      --upgrade-constraint-policy string        controls whether the package upgrade path(s) defined in the catalog are enforced. One of [CatalogProvided SelfCertified], (default CatalogProvided)
  -v, --version string                          version (or version range) in semver format to limit the allowable package versions to. If used with '--channel', only package versions meeting both constraints will be considered.
```

The flags allow for setting most mutable fields:
- `--ignore-unset`: Sets the behavior of unspecified or empty flags, whether they should be ignored, preserving the current value on the resource, or treated as valid and used to set the field values to their default value.
- `-v`, `--version`: A version or version range to restrict search for a version upgrade. If specified along with `--channels`, only versions in the version range belonging to one or more of the channels specified will be allowed.
  Valid version range format examples:
  - Exact: `--version 1.2.3`
  - Range: `--version ">=1.0.0 <2.0.0"`
  - Wildcard: `--version 1.2.x` (any patch version of 1.2)
  - Minimum: `--version ">=1.5.0"`
- `-c`, `--channels`: An optional list of channels within a package to restrict searches for updates. If empty or unspecified, no channel restrictions apply while searching for valid package versions for extension updates.
- `--upgrade-constraint-policy`: Specifies upgrade selection behavior. Valid values: `CatalogProvided|SelfCertified`. `SelfCertified` can be used to override upgrade graphs within a catalog and upgrade to any version at the risk of using non-standard upgrade paths. `CatalogProvided` restricts upgrades to standard paths between versions explicitly allowed within the `ClusterCatalog`.
- `--labels`: Additional labels to add to the `ClusterExtension` as `key=value` pairs. This flag may be specified multiple times. Setting the value of a label to an empty string deletes the label from the resource.
- `--dry-run`: Generate the manifest that would be applied with the command without actually applying it to the cluster.
- `--output`: The format for displaying manifests if `--dry-run` is specified.
- `--catalog-selector`: Limit the sources that the package specified by the ClusterExtension can be installed from to ClusterCatalogs matching the provided label selector.
- `--crd-upgrade-safety-enforcement`: configures pre-flight CRD Upgrade safety checks. If set to `Strict`, an upgrade will be blocked if it has breaking changes to a CRD on cluster. If set to `None`, this pre-flight check is skipped, which may cause unsafe changes during installs and upgrades.

To update specific fields on an extension, like adding a new label or updating desired version range, the required flag may be used on its own:
```bash
$ kubectl operator olmv1 update extension myex --labels newlabel=newkey --labels labeltoremove=
$ kubectl operator olmv1 update extension myex --version 1.2.x
```

To reset a specific field to its default on an extension, the value needs to be provided or all existing fields must be specified with `--ignore-unset`.
```bash
$ kubectl operator olmv1 update extension --upgrade-constraint-policy CatalogProvided
$ kubectl operator olmv1 update extension --ignore-unset=false --version=1.0.x --channels=stable,candidate --labels existing1=labelvalue1 --labels existing2=labelvalue2
```

<br/>
<br/>

---

## olmv1 get
Display status information of `olmv1` resources.

```bash
Display one or many resource(s)

Usage:
  operator olmv1 get [command]

Available Commands:
  catalog     Display one or many installed catalogs
  extension   Display one or many installed extensions
```
<br/>

### olmv1 get catalog
Display status information of all `ClusterCatalogs`.

```bash
Display one or many installed catalogs

Usage:
  operator olmv1 get catalog [catalog_name] [flags]

Aliases:
  catalog, catalogs

Flags:
  -o, --output string     output format. One of: (json, yaml)
  -l, --selector string   selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.
```

The flags allow for limiting or formatting output:
- `--output`: The format for displaying the resources. Valid values: json, yaml.
- `--selector`: Limit the resources listed to those matching the provided label selector.

```bash
$ kubectl operator olmv1 get catalog
NAME           AVAILABILITY  PRIORITY  LASTUNPACKED  SERVING AGE
operatorhubio  Available     0         44m           True    44m
```
<br/>

### olmv1 get extension
Display status information of installed `ClusterExtensions`.

```bash
Display one or many installed extensions

Usage:
  operator olmv1 get extension [extension_name] [flags]

Aliases:
  extension, extensions [extension_name]

Flags:
  -o, --output string     output format. One of: (json, yaml)
  -l, --selector string   selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.
```

The flags allow for limiting or formatting output:
- `--output`: The format for displaying the resources. Valid values: json, yaml.
- `--selector`: Limit the resources listed to those matching the provided label selector.

```bash
$ kubectl operator olmv1 get extension
NAME            INSTALLED BUNDLE            VERSION   SOURCE TYPE           INSTALLED   PROGRESSING   AGE
test-operator   prometheusoperator.0.47.0   0.47.0    Community Operators Index True        False         44m
```

## olmv1 search
Search available sources for packages or versions. Currently supports searching ClusterCatalogs

```bash
Search one or all available catalogs for packages or versions

Usage:
  operator olmv1 search [command]

Available Commands:
  catalog     Search catalogs for installable operators matching parameters
```
<br/>

### olmv1 search catalog
Search one or all available catalogs for packages or versions

```bash
kubectl-operator olmv1 search catalog --help
Search catalogs for installable operators matching parameters

Usage:
  operator olmv1 search catalog [flags]

Aliases:
  catalog, catalogs

Flags:
      --catalog string              name of the catalog to search. If not provided, all available catalogs are searched.
      --catalogd-namespace string   namespace for the catalogd controller (default "olmv1-system")
      --list-versions               list all versions available for each package
  -o, --output string               output format. One of: (yaml|json)
      --package string              search for package by name. If empty, all available packages will be listed
  -l, --selector string             selector (label query) to filter catalogs on, supports '=', '==', and '!='
      --timeout string              timeout for fetching catalog contents (default "5m")
```

The flags allow for limiting or formatting output:
- `--catalog`: When non-empty, limits the listed packages or bundles to only those from the catalog provided through this flag
- `--selector`: When non-empty, limits the listed packages or bundles to only the ones from ClusterCatalogs matching the label selector provided through this flag.
- `--timeout`: Time to wait before aborting the command
- `--catalogd-namespace`: By default, the catalogd-controller is installed in `olmv1-system`. If the catalogd-controller's service is present in another namespace, it can be specified through this flag.
- `--list-versions`: By default, the search command shows the package name, the source ClusterCatalog, the package maintainer and the valid channels available on the package, if any. If this flag is specified, it lists the versions available for each package instead of listing channels.
- `--package`: If non-empty, limit the listed packages and bundles to only those with the package name specified by this flag.
- `--output`: This flag allows the output to be provided in a specific format. Currently support yaml, json. If empty, provides a simplified table of packages instead.

Minimal example:
```bash
$ kubectl operator olmv1 search catalog

PACKAGE                                   CATALOG        PROVIDER  CHANNELS
accuknox-operator                         operatorhubio            stable
ack-acm-controller                        operatorhubio            alpha
ack-acmpca-controller                     operatorhubio            alpha
...
```

Full examples:
Restricting to a specific package:
```bash
$ kubectl operator olmv1 search catalog --package etcd

PACKAGE  CATALOG        PROVIDER  CHANNELS
etcd     operatorhubio            alpha,clusterwide-alpha,singlenamespace-alpha
```

Listing versions from a package:
```bash
$ kubectl operator olmv1 search catalog --package etcd --list-versions

PACKAGE  CATALOG        PROVIDER     VERSION
etcd     operatorhubio  CoreOS, Inc  0.9.4
etcd     operatorhubio  CoreOS, Inc  0.9.4-clusterwide
etcd     operatorhubio  CoreOS, Inc  0.9.2
etcd     operatorhubio  CoreOS, Inc  0.9.2-clusterwide
etcd     operatorhubio  CoreOS, Inc  0.9.0
etcd     operatorhubio  CoreOS, Inc  0.6.1
```

#### Specific output formats:
##### YAML output format:
```bash
$kubectl operator olmv1 search catalog --package vault-helm -o yaml
---
defaultChannel: alpha
icon:
  base64data: PHN2ZyBp=
  mediatype: image/svg+xml
name: vault-helm
schema: olm.package
---
entries:
- name: vault-helm.v0.0.1
- name: vault-helm.v0.0.2
  replaces: vault-helm.v0.0.1
name: alpha
package: vault-helm
schema: olm.channel
---
image: quay.io/brejohns/vault-helm:0.0.2
name: vault-helm.v0.0.1
package: vault-helm
properties:
- type: olm.gvk
  value:
    group: vault.sdbrett.com
    kind: Vault
    version: v1alpha1
- type: olm.package
  value:
    packageName: vault-helm
    version: 0.0.1
relatedImages:
- image: gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
  name: ""
- image: quay.io/brejohns/vault-helm:0.0.1
  name: ""
schema: olm.bundle
---
image: quay.io/brejohns/vault-helm:0.0.2
name: vault-helm.v0.0.2
package: vault-helm
properties:
- type: olm.gvk
  value:
    group: vault.sdbrett.com
    kind: Vault
    version: v1alpha1
- type: olm.package
  value:
    packageName: vault-helm
    version: 0.0.2
- type: olm.csv.metadata
  value:
    annotations:
      alm-examples: |-
        [
          {
            "apiVersion": "vault.sdbrett.com/v1alpha1",
            "kind": "Vault",
          }
        ]
      capabilities: Basic Install
      categories: Security
      containerImage: quay.io/brejohns/vault-helm:0.0.2
      createdAt: "2021-01-27 09:00:00"
      description: Use Helm to Deploy and manage Hashicorp Vault
      operatorhub.io/ui-metadata-max-k8s-version: "1.21"
      operators.operatorframework.io/builder: operator-sdk-v1.2.0
      operators.operatorframework.io/project_layout: helm.sdk.operatorframework.io/v1
      repository: https://github.com/SDBrett/vault-helm
    apiServiceDefinitions: {}
    crdDescriptions:
      owned:
      - description: Provides Vault-Helm operators with values for the Vault Helm
          chart
        displayName: Vault Helm
        kind: Vault
        name: vaults.vault.sdbrett.com
        version: v1alpha1
    description: Operator for deployment and management of Hashicorp Vault instances
      based on the Helm chart
    displayName: Vault Helm
    installModes:
    - supported: true
      type: OwnNamespace
    - supported: false
      type: SingleNamespace
    - supported: false
      type: MultiNamespace
    - supported: true
      type: AllNamespaces
    keywords:
    - Helm
    - Vault
    - Hashicorp
    maintainers:
    - email: brett@sdbrett.com
      name: Brett Johnson
    maturity: alpha
    provider:
      name: SDBrett
      url: sdbrett.com
relatedImages:
- image: gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
  name: ""
- image: quay.io/brejohns/vault-helm:0.0.2
  name: ""
schema: olm.bundle
---
```

##### JSON output format:
```json
$ kubectl operator olmv1 search catalog --package vault-helm -o json
{
    "schema": "olm.package",
    "name": "vault-helm",
    "defaultChannel": "alpha",
    "icon": {
        "base64data": "PHN2ZyBpZD0i=",
        "mediatype": "image/svg+xml"
    }
}
{
    "schema": "olm.channel",
    "name": "alpha",
    "package": "vault-helm",
    "entries": [
        {
            "name": "vault-helm.v0.0.1"
        },
        {
            "name": "vault-helm.v0.0.2",
            "replaces": "vault-helm.v0.0.1"
        }
    ]
}
{
    "schema": "olm.bundle",
    "name": "vault-helm.v0.0.1",
    "package": "vault-helm",
    "image": "quay.io/brejohns/vault-helm:0.0.1",
    "properties": [
        {
            "type": "olm.gvk",
            "value": {
                "group": "vault.sdbrett.com",
                "kind": "Vault",
                "version": "v1alpha1"
            }
        },
        {
            "type": "olm.package",
            "value": {
                "packageName": "vault-helm",
                "version": "0.0.1"
            }
        }
    ],
    "relatedImages": [
        {
            "name": "",
            "image": "gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0"
        },
        {
            "name": "",
            "image": "quay.io/brejohns/vault-helm:0.0.1"
        },
    ]
}
{
    "schema": "olm.bundle",
    "name": "vault-helm.v0.0.2",
    "package": "vault-helm",
    "image": "quay.io/brejohns/vault-helm:0.0.2",
    "properties": [
        {
            "type": "olm.gvk",
            "value": {
                "group": "vault.sdbrett.com",
                "kind": "Vault",
                "version": "v1alpha1"
            }
        },
        {
            "type": "olm.package",
            "value": {
                "packageName": "vault-helm",
                "version": "0.0.2"
            }
        },
        {
            "type": "olm.csv.metadata",
            "value": {
                "annotations": {
                    "alm-examples": "[\n{\n\"apiVersion\": \"vault.sdbrett.com/v1alpha1\",\n\"kind\" \"Vault\"}\n]",
                    "capabilities": "Basic Install",
                    "categories": "Security",
                    "containerImage": "quay.io/brejohns/vault-helm:0.0.2",
                    "createdAt": "2021-01-27 09:00:00",
                    "description": "Use Helm to Deploy and manage Hashicorp Vault",
                    "operatorhub.io/ui-metadata-max-k8s-version": "1.21",
                    "operators.operatorframework.io/builder": "operator-sdk-v1.2.0",
                    "operators.operatorframework.io/project_layout": "helm.sdk.operatorframework.io/v1",
                    "repository": "https://github.com/SDBrett/vault-helm"
                },
                "apiServiceDefinitions": {},
                "crdDescriptions": {
                    "owned": [
                        {
                            "description": "Provides Vault-Helm operators with values for the Vault Helm chart",
                            "displayName": "Vault Helm",
                            "kind": "Vault",
                            "name": "vaults.vault.sdbrett.com",
                            "version": "v1alpha1"
                        }
                    ]
                },
                "description": "Operator for deployment and management of Hashicorp Vault instances based on the Helm chart",
                "displayName": "Vault Helm",
                "installModes": [
                    {
                        "supported": true,
                        "type": "OwnNamespace"
                    },
                    {
                        "supported": false,
                        "type": "SingleNamespace"
                    },
                    {
                        "supported": false,
                        "type": "MultiNamespace"
                    },
                    {
                        "supported": true,
                        "type": "AllNamespaces"
                    }
                ],
                "keywords": [
                    "Helm",
                    "Vault",
                    "Hashicorp"
                ],
                "maintainers": [
                    {
                        "email": "brett@sdbrett.com",
                        "name": "Brett Johnson"
                    }
                ],
                "maturity": "alpha",
                "provider": {
                    "name": "SDBrett",
                    "url": "sdbrett.com"
                }
            }
        }
    ],
    "relatedImages": [
        {
            "name": "",
            "image": "gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0"
        },
        {
            "name": "",
            "image": "quay.io/brejohns/vault-helm:0.0.2"
        },
    ]
}
```

## Recommended Workflows

### Adding a catalog:
  1. **Validate inputs**:
     - Verify catalog name doesn't already exist: `kubectl operator olmv1 get catalog <catalog_name>`
     - Should return: not found (error is expected)
  2. **Create catalog**:
```bash
kubectl operator olmv1 create catalog <catalog_name> <image>
```
  - Expected output: catalog "<catalog_name>" created
  - If no output or different message: creation failed
  3. Poll for serving status (if --available=true or default):
  kubectl operator olmv1 get catalog <name>
    - Poll every 15-30 seconds
    - Wait until SERVING: True or timeout after 5 minutes. This may be slightly greater for very large catalogs.
    - If SERVING stays False, check kubectl operator olmv1 get catalog <name> -o yaml for error conditions
  4. Verify packages available (optional):
  kubectl operator olmv1 search catalog --catalog <name>
    - Should show packages from the catalog
    - If no packages, catalog may be empty or invalid

### Installing a package:
  For reliable installation, follow this sequence:
  1. **Search for the package**:
```bash
$ kubectl operator olmv1 search catalog --package <package-name>
```
```bash
$ kubectl operator olmv1 search catalog --package <package-name> --list-versions
```
     Verify the package exists and note available channels/versions.
     If package-name is not known, search the catalog for a potential match:
```bash
$ kubectl operator olmv1 search catalog
```
  2. Check catalogs are serving:
  kubectl operator olmv1 get catalog
  2. Ensure at least one catalog shows SERVING: True.
  3. Ensure a ServiceAccount with the required permissions for installing the operator. May provide blanket permissions on dry-runs.
  4. Install the extension:
```bash
$ kubectl operator olmv1 install extension <extension_name> --namespace <namespace> --package-name <package-name> --service-account <service_account_name>
```
  5. Verify installation:
```bash
$ kubectl operator olmv1 get extension <extension_name>
```
    Wait for INSTALLED: True.

