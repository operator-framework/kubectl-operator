# kubectl operator

`kubectl operator` is a kubectl plugin that functions as a package manager
for Operators in your cluster. It simplifies adding and removing Operator
catalogs, and it has familiar commands for installing, uninstalling, and
listing available and installed Operators.

kubectl operator can be used for both OLM VO and V1.

To see the OLM V1 related commands , run

```
kubectl operator olmv1 --help

```

> **NOTE**: This plugin requires Operator Lifecycle Manager to be installed in your
cluster. See the OLM installation instructions [here](https://olm.operatorframework.io/docs/getting-started/)

## Demo

### OLM V0

![asciicast](assets/demo/demo.gif)

## Install

The `kubectl operator` plugin is distributed via [`krew`](https://krew.sigs.k8s.io/). To install it, run the following:
```console
kubectl krew install operator
```
