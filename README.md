## Nginx Conf Generator
[![CI](https://github.com/bilalcaliskan/nginx-conf-generator/workflows/CI/badge.svg?event=push)](https://github.com/bilalcaliskan/nginx-conf-generator/actions?query=workflow%3ACI)
[![Go Report Card](https://goreportcard.com/badge/github.com/bilalcaliskan/nginx-conf-generator)](https://goreportcard.com/report/github.com/bilalcaliskan/nginx-conf-generator)

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

#### Single cluster
Below flags are the keys to communicate with cluster:
```
kubeConfigPaths := flag.String("kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "config"),
		"comma seperated list of kubeconfig file paths to access with the cluster")
// single worker node ip address for each cluster. order of ip addresses must be same with kubeConfigPaths[]
workerNodeIps := flag.String("workerNodeIps", "192.168.0.201", "comma seperated ip " +
		"address of the worker nodes to reach the services over NodePort")
```

#### Multi cluster
Provide a comma seperated list of arguments to the flag `-kubeConfigPaths` and `workerNodeIps` with the same order.

### Download

#### Binary
Binary can be downloaded from [Releases](https://github.com/bilalcaliskan/nginx-conf-generator/releases) page.

#### Docker
Docker image can be downloaded with below command:
```shell
$ docker run bilalcaliskan/nginx-conf-generator:latest
```

### Development
This project requires below tools while developing:
- [pre-commit](https://pre-commit.com/)
- [golangci-lint](https://golangci-lint.run/usage/install/) - required by [pre-commit](https://pre-commit.com/)
