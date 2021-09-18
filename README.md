# Nginx Conf Generator
[![CI](https://github.com/bilalcaliskan/nginx-conf-generator/workflows/CI/badge.svg?event=push)](https://github.com/bilalcaliskan/nginx-conf-generator/actions?query=workflow%3ACI)
[![Go Report Card](https://goreportcard.com/badge/github.com/bilalcaliskan/nginx-conf-generator)](https://goreportcard.com/report/github.com/bilalcaliskan/nginx-conf-generator)
[![codecov](https://codecov.io/gh/bilalcaliskan/nginx-conf-generator/branch/master/graph/badge.svg)](https://codecov.io/gh/bilalcaliskan/nginx-conf-generator)
[![Release](https://img.shields.io/github/release/bilalcaliskan/nginx-conf-generator.svg)](https://github.com/bilalcaliskan/nginx-conf-generator/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/bilalcaliskan/nginx-conf-generator)](https://github.com/bilalcaliskan/nginx-conf-generator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

That tool uses [client-go](https://github.com/kubernetes/client-go) to communicate with multi Kubernetes clusters and
gets the port of NodePort type service which contains specific annotation. Then modifies
the Nginx configuration and reloads the Nginx process.


## Prerequisites
nginx-conf-generator uses the kubeconfig file for authentication and authorization with Kubernetes cluster.
You should ensure that given kubeconfig file has read only access on the target cluster.

Also nginx-conf-generator needs to reload nginx process when necessary, you must run it with root user.

## Configuration
nginx-conf-generator can be customized with several command line arguments:
```
--kubeConfigPaths       comma separated list of kubeconfig file paths to access with the cluster, defaults to ~/.kube.config
--workerNodeLabel       label to specify worker nodes, defaults to node-role.k8s.io/worker=
--customAnnotation      annotation to specify selectable services, defaults to nginx-conf-generator/enabled
--templateInputFile     input path of the template file, defaults to ./resources/default.conf.tmpl
--templateOutputFile    output path of the template file, defaults to /etc/nginx/sites-enabled/default
--metricsPort           port of the metrics server, defaults to 5000
--writeTimeoutSeconds   write timeout of the metrics server, defaults to 10
--readTimeoutSeconds    read timeout of the metrics server, defaults to 10
--metricsEndpoint       endpoint to provide prometheus metrics, defaults to /metrics
```

> That tool should be run on a Linux host and the user who runs the binary file nginx-conf-generator
should have permissions to edit **--templateOutputFile** file and reload nginx process using below command:
> ```shell
> $ nginx -s reload
> ```

> If you want to cover multiple kubernetes clusters, add comma seperated list of kubeconfig paths with **--kubeConfigPaths** argument.

## Installation
Binary can be downloaded from [Releases](https://github.com/bilalcaliskan/nginx-conf-generator/releases) page.

After then, you can simply run binary by providing required command line arguments:
```shell
$ ./nginx-conf-generator --kubeConfigPaths ~/.kube/config1,~/.kube/config2 --customAnnotation nginx-conf-generator/enabled
```

## Development
This project requires below tools while developing:
- [Golang 1.16](https://golang.org/doc/go1.16)
- [pre-commit](https://pre-commit.com/)
- [golangci-lint](https://golangci-lint.run/usage/install/) - required by [pre-commit](https://pre-commit.com/)

## License
Apache License 2.0
