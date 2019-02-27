[![Build Status](https://dev.azure.com/dgkanatsios/AksNodePublicIPController/_apis/build/status/AksNodePublicIPController-CI?branchName=master)](https://dev.azure.com/dgkanatsios/AksNodePublicIPController/_build/latest?definitionId=1&branchName=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/AksNodePublicIPController)](https://goreportcard.com/report/github.com/dgkanatsios/AksNodePublicIPController)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AksNodePublicIPController)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-beta-yellow.svg)

# AksNodePublicIPController

[Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/) does not currently have a way to automatically assign Public IPs to worker nodes/virtual machines. This project aims to solve this problem by utilizing a custom Kubernetes controller (based on [sample-controller](https://github.com/kubernetes/sample-controller)) and using [Azure SDK for Go](https://docs.microsoft.com/en-us/go/azure/). The ID for the new Public IPs is always "ipconfig-" + name of the Node/Virtual Machine. It also assigns a Kubernetes Label to the Node, with name "HasPublicIP" and value "true".

### Deployment

If you have an RBAC enabled cluster, just run:

```bash
kubectl create -n kube-system -f https://raw.githubusercontent.com/dgkanatsios/AksNodePublicIPController/master/deploy.yaml
# this gets created into *kube-system* namespace, feel free to change it on the deploy.yaml
```

else, run:

```bash
kubectl create -f https://raw.githubusercontent.com/dgkanatsios/AksNodePublicIPController/master/deploy-no-rbac.yaml
```

### Alternatives

If you're looking for a non-Kubernetes native solution, you should check out the [AksNodePublicIP](https://github.com/dgkanatsios/AksNodePublicIP) project, it uses [Azure Functions](https://functions.azure.com) and [Azure Event Grid](https://azure.microsoft.com/en-us/services/event-grid/) technologies.