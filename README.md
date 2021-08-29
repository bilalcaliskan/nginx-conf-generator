## Nginx Conf Generator
[![CI](https://github.com/bilalcaliskan/nginx-conf-generator/workflows/CI/badge.svg?event=push)](https://github.com/bilalcaliskan/nginx-conf-generator/actions?query=workflow%3ACI)
[![Go Report Card](https://goreportcard.com/badge/github.com/bilalcaliskan/nginx-conf-generator)](https://goreportcard.com/report/github.com/bilalcaliskan/nginx-conf-generator)
[![codecov](https://codecov.io/gh/bilalcaliskan/nginx-conf-generator/branch/master/graph/badge.svg)](https://codecov.io/gh/bilalcaliskan/nginx-conf-generator)

That tool uses `client-go` to communicate with multi Kubernetes clusters and picks the NodePort port
of services which is a NodePort type service and contains specific annotation. Default annotation can
be changed with above command line flag:
```
customAnnotation := flag.String("customAnnotation", "nginx-conf-generator/enabled", "annotation to specify " +
		"selectable services")
```

That tool should be run on an Ubuntu host and the user who runs the binary file nginx-conf-generator
should have permissions to edit below file and reload nginx service:
```
templateOutputFile := flag.String("templateOutputFile", "/etc/nginx/sites-enabled/default", "output " +
		"path of the template file")
```

Tool uses the kubeconfig file for authentication and authorization with Kubernetes cluster. You should consider
create only required role and rolebinding for the tool.

Then modifies the `templateOutputFile(defaults to /etc/nginx/sites-enabled/default)` and reloads the Nginx process.

### Configuration
nginx-conf-generator can be customized with several command line arguments:
```shell
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

> If you want to cover multiple kubernetes clusters, add comma seperated list of kubeconfig paths with **--kubeConfigPaths** argument.

### Download
Binary can be downloaded from [Releases](https://github.com/bilalcaliskan/nginx-conf-generator/releases) page.

After then, you can simply run binary by providing required command line arguments:
```shell
$ ./nginx-conf-generator --kubeConfigPaths ~/.kube/config1,~/.kube/config2 --customAnnotation nginx-conf-generator/enabled
```

> Since nginx-conf-generator needs to reload nginx systemd service when necessary, you must run it with root user.

### Development
This project requires below tools while developing:
- [pre-commit](https://pre-commit.com/)
- [golangci-lint](https://golangci-lint.run/usage/install/) - required by [pre-commit](https://pre-commit.com/)
