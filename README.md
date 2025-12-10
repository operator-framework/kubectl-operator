# kubectl operator

`kubectl operator` is a kubectl plugin that functions as a package manager
for Operators in your cluster. It simplifies adding and removing Operator
catalogs, and it has familiar commands for installing, uninstalling, and
listing available and installed Operators.


> **NOTE**: This plugin requires Operator Lifecycle Manager to be installed in your
cluster. See the OLM installation instructions [here](https://olm.operatorframework.io/docs/getting-started/)

## Demo

![asciicast](assets/demo/demo.gif)

## Install

The `kubectl operator` plugin is distributed via [`krew`](https://krew.sigs.k8s.io/). To install it, run the following:
```console
kubectl krew install operator
```

# Installing from source

Prerequisites:
- [go 1.23.4+](https://go.dev/dl/)
- `GOPATH` environment variable must be set.

To build the latest version from the repository, run `make install`. This should install the kubectl-operator plugin to the `$GOPATH`/bin directory. 

To enable `kubectl` to discover the plugin, ensure that the '$GOPATH'/bin directory is either part of the '$PATH' environment variable, or that the newly installed `kubectl-operator` is moved to a directory in the `$PATH`.

You should see that the plugin has successfully installed:
```bash
$ kubectl operator

Manage operators in a cluster from the command line.

kubectl operator helps you manage operator installations in your
cluster. It can install and uninstall operator catalogs, list
operators available for installation, and install and uninstall
operators from the installed catalogs.

Usage:
  operator [command]

Available Commands:
  catalog        Manage operator catalogs
  completion     Generate the autocompletion script for the specified shell
  describe       Describe an operator
  help           Help about any command
  install        Install an operator
  list           List installed operators
  list-available List operators available to be installed
  list-operands  List operands of an installed operator
  olmv1          Manage OLMv1 extensions and catalogs
  uninstall      Uninstall an operator and operands
  upgrade        Upgrade an operator
  version        Print version information

Flags:
  -h, --help               help for operator
  -n, --namespace string   If present, namespace scope for this CLI request
      --timeout duration   The amount of time to wait before giving up on an operation. (default 1m0s)

Use "operator [command] --help" for more information about a command.
```
