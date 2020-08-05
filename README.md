# kubectl operator

`kubectl operator` is a kubectl plugin that functions as a package manager
for Operators in your cluster. It simplifies adding and removing Operator
catalogs, and it has familiar commands for installing, uninstalling, and
listing available and installed Operators.


> **NOTE**: This plugin requires Operator Lifecycle Manager to be installed in your
cluster. See the OLM installation instructions [here](https://olm.operatorframework.io/docs/getting-started/)

## Demo

![asciicast](assets/asciinema-demo.gif)

## Install

Official Kubectl plugin documentation includes [plugin install documentation](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/#installing-kubectl-plugins). In short, find the required release on the [releases page](https://github.com/operator-framework/kubectl-operator/releases), decompress and place the kubectl-operator binary in your path. Kubectl will figure it out from there. 
