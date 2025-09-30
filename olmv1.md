# Kubectl Operator OLMv1 Commands

Most of the kubectl operator plugin subcommands are aimed toward managing OLMv0 operators.

[OLMv1](https://github.com/operator-framework/operator-controller/) has two main on-cluster CRDs, both of which can be managed by the kubectl operator `olmv1` subcommand. These are:
- [`ClusterCatalogs`](https://github.com/operator-framework/operator-controller/blob/main/api/v1/clustercatalog_types.go): A CR used to provide a curated collection of packages for users to install and upgrade from, backed by an image reference to a File Based Catalog (FBC) 
- [`ClusterExtensions`](https://github.com/operator-framework/operator-controller/blob/main/api/v1/clusterextension_types.go): A CR expressing the current status and desired state of a package on cluster. It must provide the package name to be installed from available `ClusterCatalogs`, but may also specify restrictions on install and upgrade discovery.

Within the repository, these are defined in `internal/cmd/olmv1.go`, which in turn references `internal/cmd/internal/olmv1` and `internal/pkg/v1`.

```bash
$ kubectl operator olmv1 --help

Manage extensions via `olmv1` in a cluster from the command line.

Usage:
  operator olmv1 [command]

Available Commands:
  create      Create a resource
  delete      Delete a resource
  get         Display one or many resource(s)
  install     Install a resource
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
      --available                          true means that the catalog should be active and serving data (default true)
      --cleanup-timeout duration           the amount of time to wait before cancelling cleanup after a failed creation attempt (default 1m0s)
      --labels stringToString              labels that will be added to the catalog (default [])
      --priority int32                     priority determines the likelihood of a catalog being selected in conflict scenarios
      --source-poll-interval-minutes int   catalog source polling interval [in minutes] (default 10)
```

The flags allow for setting most mutable fields:
- `--available`: Sets whether the `ClusterCatalog` should be actively serving and make its contents available on cluster. Default: true.
- `--cleanup-timeout`: If a `ClusterCatalog` creation attempt fails due to the resource never becoming healthy, `olmv1` cleans up by deleting the failed resource, with a timeout specified by `--cleanup-timeout`. Default: 1 minute (1m)
- `--labels`: Additional labels to add to the newly created `ClusterCatalog` as `key=value` pairs. This flag may be specified multiple times.
- `--priority`: Integer priority used for ordering `ClusterCatalogs` in case two extension packages have the same name across `ClusterCatalogs`, with a higher value indicating greater relative priority. Default: 0
- `--source-poll-interval-minutes`: The polling interval to check for changes if the `ClusterCatalog` source image provided is not a digest based image, i.e, if it is referenced by tag. Set to 0 to disable polling. Default: 10

The command requires at minimum a resource name and image reference:
```bash
$ kubectl operator olmv1 create catalog mycatalog myorg/mycatalogrepo:tag
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
  -c, --channels strings           channels which would be used for getting updates, e.g, --channels "stable,dev-preview,preview"
  -d, --cleanup-timeout duration   the amount of time to wait before cancelling cleanup after a failed creation attempt (default 1m0s)
  -n, --namespace string           namespace to install the operator in
  -p, --package-name string        package name of the operator to install
  -s, --service-account string     service account name to use for the extension installation (default "default")
  -v, --version string             version (or version range) from which to resolve bundles
```

The flags allow for setting most mutable fields:
- **`-n`, `--namespace`**: The namespace to install any namespace-scoped resources created by the `ClusterExtension`. <span style="color:red">**Required**</span>
- **`-p`, `--package-name`**: Name of the package to install. <span style="color:red">**Required**</span>
- `-c`, `--channels`: An optional list of channels within a package to restrict searches for an installable version to. 
- `-d`, `--cleanup-timeout`: If a `ClusterExtension` creation attempt fails due to the resource never becoming healthy, `olmv1` cleans up by deleting the failed resource, with a timeout specified by `--cleanup-timeout`. Default: 1 minute (1m)
- `-s`, `--service-account`: Name of the service account present in the namespace specified by `--namespace` to use for creating and managing resources for the new `ClusterExtension`.
- `-v`, `--version`: A version or version range to restrict search for an installable version to.
<br/>
<br/>

The command requires at minimum a resource name, `--namespace` and `--package-name`:
```bash
$ kubectl operator olmv1 install extension myex --namespace myns --package-name prometheus-operator
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
      --all    delete all catalogs
```

The command requires exactly one of a resource name or the `--all` flag:
```bash
$ kubectl operator olmv1 delete catalog mycatalog
$ kubectl operator olmv1 delete catalog --all
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
  -a, --all    delete all extensions
```

The command requires exactly one of a resource name or the `--all` flag:
```bash
$ kubectl operator olmv1 delete extension myex
$ kubectl operator olmv1 delete extension --all
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
      --availability-mode string           available means that the catalog should be active and serving data
      --ignore-unset                       when enabled, any unset flag value will not be changed. Disabling this flag replaces all other unset or empty values with a default value, overwriting any values on the existing CR (default true)
      --image string                       Image reference for the catalog source. Leave unset to retain the current image.
      --labels stringToString              labels that will be added to the catalog (default [])
      --priority int32                     priority determines the likelihood of a catalog being selected in conflict scenarios
      --source-poll-interval-minutes int   catalog source polling interval [in minutes]. Set to 0 or -1 to remove the polling interval. (default 5)
```

The flags allow for setting most mutable fields:
- `--ignore-unset`: Sets the behavior of unspecified or empty flags, whether they should be ignored, preserving the current value on the resource, or treated as valid and used to set the field values to their default value.
- `--availablity-mode`: Sets whether the `ClusterCatalog` should be actively serving and making its contents available on cluster. Valid values: `Available`|`Unavailable`.
- `--image`: Update the image reference for the `ClusterCatalog`.
- `--labels`: Additional labels to add to the `ClusterCatalog` as `key=value` pairs. This flag may be specified multiple times. Setting the value of a label to an empty string deletes the label from the resource.
- `--priority`: Integer priority used for ordering `ClusterCatalogs` in case two extension packages have the same name across `ClusterCatalogs`, with a higher value indicating greater relative priority.
- `--source-poll-interval-minutes`: The polling interval to check for changes if the `ClusterCatalog` source image provided is not a digest based image, i.e, if it is referenced by tag. Set to 0 or -1 to disable polling.
<br/>

To update specific fields on a catalog, like adding a new label or setting availability, the required flag may be used on its own:
```bash
$ kubectl operator olmv1 update catalog mycatalog --label newlabel=newkey --label labeltoremove=
$ kubectl operator olmv1 update catalog --availability-mode Available
```

To reset a specific field on a catalog to its default, the value needs to be provided or all existing fields must be specified with `--ignore-unset`.
```bash
$ kubectl operator olmv1 update catalog mycatalog --availability-mode Available
$ kubectl operator olmv1 update catalog mycatalog --ignore-unset=false --priority=10 --source-poll-interval-minutes=-1 --image=myorg/mycatalogrepo:tag --labels existing1=labelvalue1 --labels existing2=labelvalue2
```

### olmv1 update extension

Update supported mutable fields on a `ClusterExtension` specified by name.

```bash
Update an extension

Usage:
  operator olmv1 update extension <extension> [flags]

Flags:
      --channels stringArray               desired channels for extension versions. AND operation with version. If empty or not specified, all available channels will be taken into consideration
      --ignore-unset                       when enabled, any unset flag value will not be changed. Disabling this flag replaces all other unset or empty values with a default value, overwriting any values on the existing CR (default true)
      --labels stringToString              labels that will be set on the extension (default [])
      --selector string                    filters the set of catalogs used in the bundle selection process. Empty means that all catalogs will be used in the bundle selection process
      --upgrade-constraint-policy string   controls whether the upgrade path(s) defined in the catalog are enforced. One of CatalogProvided|SelfCertified, Default: CatalogProvided
      --version string                     desired extension version (single or range) in semVer format. AND operation with channels
```

The flags allow for setting most mutable fields:
- `--ignore-unset`: Sets the behavior of unspecified or empty flags, whether they should be ignored, preserving the current value on the resource, or treated as valid and used to set the field values to their default value.
- `-v`, `--version`: A version or version range to restrict search for a version upgrade.
- `-c`, `--channels`: An optional list of channels within a package to restrict searches for updates. If empty or unspecified, no channel restrictions apply while searching for valid package versions for extension updates.
- `--upgrade-constraint-policy`: Specifies upgrade selection behavior. Valid values: `CatalogProvided|SelfCertified`. `SelfCertified` can be used to override upgrade graphs within a catalog and upgrade to any version at the risk of using non-standard upgrade paths. `CatalogProvided` restricts upgrades to standard paths between versions explicitly allowed within the `ClusterCatalog`.
- `--labels`: Additional labels to add to the `ClusterExtension` as `key=value` pairs. This flag may be specified multiple times. Setting the value of a label to an empty string deletes the label from the resource.

To update specific fields on an extension, like adding a new label or updating desired version range, the required flag may be used on its own:
```bash
$ kubectl operator olmv1 update extension myex --label newlabel=newkey --label labeltoremove=
$ kubectl operator olmv1 update extension myex --version 1.2.x
```

To reset a specific field to its default on an extension, the value needs to be provided or all existing fields must be specified with `--ignore-unset`.
```bash
$ kubectl operator olmv1 update catalog --upgrade-constraint-policy CatalogProvided
$ kubectl operator olmv1 update catalog --ignore-unset=false --version=1.0.x --channels=stable,candidate --labels existing1=labelvalue1 --labels existing2=labelvalue2
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
```

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
```

```bash
$ kubectl operator olmv1 get extension
NAME            INSTALLED BUNDLE            VERSION   SOURCE TYPE           INSTALLED   PROGRESSING   AGE
test-operator   prometheusoperator.0.47.0   0.47.0    Community Operators Index True        False         44m
```