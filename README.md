[![Build Status](https://dev.azure.com/dgkanatsios/AksNodePublicIPController/_apis/build/status/AksNodePublicIPController-CI?branchName=master)](https://dev.azure.com/dgkanatsios/AksNodePublicIPController/_build/latest?definitionId=1&branchName=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/AksNodePublicIPController)](https://goreportcard.com/report/github.com/dgkanatsios/AksNodePublicIPController)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AksNodePublicIPController)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-beta-yellow.svg)

# AksNodePublicIPController

[Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/) does not currently have a way to automatically assign Public IPs to worker nodes/virtual machines. This project aims to solve this problem by utilizing a custom Kubernetes controller (based on [sample-controller](https://github.com/kubernetes/sample-controller)) and using [Azure SDK for Go](https://docs.microsoft.com/en-us/go/azure/). The ID for the new Public IPs is always "ipconfig-" + name of the Node/Virtual Machine. It also assigns a Kubernetes Label to the Node, with name "HasPublicIP" and value "true".

## Deployment

### AKS clusters using Availability Sets

(This is probably what you're using)

If you have an RBAC enabled cluster, just run:

```bash
kubectl create -n kube-system -f https://raw.githubusercontent.com/dgkanatsios/AksNodePublicIPController/master/deploy.yaml
# this gets created into *kube-system* namespace, change it on the deploy.yaml
```

else, run:

```bash
kubectl create -f https://raw.githubusercontent.com/dgkanatsios/AksNodePublicIPController/master/deploy-no-rbac.yaml
```

#### Alternatives

If you're looking for a non-Kubernetes native solution, you should check out the [AksNodePublicIP](https://github.com/dgkanatsios/AksNodePublicIP) project, it uses [Azure Functions](https://functions.azure.com) and [Azure Event Grid](https://azure.microsoft.com/en-us/services/event-grid/) technologies.

### AKS clusters using Virtual Machine Scale Sets

If you have created an AKS cluster using Virtual Machine Scale Set (VMSS) functionality, then the process is easier, since you don't need to deploy anything. What you need to do is:

- Visit [resources.azure.com](https://resources.azure.com) to view your deployed Azure resources
- Find the resource group where your AKS resources are deployed. It should have a name like `MC_aksInstanceName_aksResourceGroupName_dataCenterLocation`
- Find and extend your VMSS information. VMSS should have a name like `aks-nodepool1-34166363-vmss`
- Edit it and add the following JSON into `ipConfigurations.properties` section ([source](https://docs.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-networking#creating-a-scale-set-with-public-ip-per-virtual-machine)):
```json
"publicIpAddressConfiguration": {
    "name": "pub1"
}
```
To better understand where to place it, check [here](https://github.com/Azure/azure-quickstart-templates/blob/master/201-vmss-public-ip-linux/azuredeploy.json#L187)
- Press `Patch` or `Put` on the UI. VMSS should now be configured so that newly created VMs get a Public IP by default
- Execute a scale out and a scale in operation on the cluster so existing VMs get a Public IP
- Run `kubectl get node -o wide` and verify that all your Nodes have got a Public IP

To debug, you should run it like:

```bash
TENANT_ID=XXX SUBSCRIPTION_ID=XXX AAD_CLIENT_ID=XXX AAD_CLIENT_SECRET=XXX LOCATION=XXX RESOURCE_GROUP=XXX go run . --kubeconfig=~/.kube/config-aksopenarena
```

after getting the env details via this Pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: busybox
spec:
  containers:
  - image: busybox
    command: ["cat","/akssp/azure.json"]
    name: busybox
    volumeMounts:
      - name: akssp
        mountPath: /akssp
  restartPolicy: Never
  volumes:
  - name: akssp
    hostPath:
      path: /etc/kubernetes
      type: Directory
```

```bash
kubect logs busybox -f
```

*Kudos to [Andreas Pohl](https://twitter.com/annonator) for the guidance with VMSS*