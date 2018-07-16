[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/AksNodePublicIPController)](https://goreportcard.com/report/github.com/dgkanatsios/AksNodePublicIPController)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AksNodePublicIPController)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-alpha-red.svg)

# AksNodePublicIPController

A project that can be deployed to an Azure Kubernetes Cluster and will allow each node to obtain a Public IP address.

### Deployment

Just run

```bash
kubectl create -f deploy.yaml https://raw.githubusercontent.com/dgkanatsios/AksNodePublicIPController/master/deploy.yaml
```