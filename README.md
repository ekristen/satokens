# satokens (aka Service Account Tokens) 

**Status:** beta

The purpose of this project is to make developing locally easier with respect to using Kubernetes Service Account
tokens for federated authentication.

Suggested Reading: [What Are Service Account Tokens](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)

## How It Works

This tool allows you to deploy a pod into a cluster's namespace. The pod is configured to have a projected volume
that adds a JWT token signed by the cluster for the associated service account. This pod serves up the token on
an http API that's available on port 44044 within the container.

This tool then offers a `mount` command that uses a fuse filesystem and port-forwarding capabilities of kubernetes.
The mount command opens a connection to the deployed container and mounts the fuse filesystem. The filesystem as a
single file called `token`. The contents of this token is the cluster generated token.

You can simply read the file at will and use during development of applications and tools as if you were running in the
cluster.

## License

Apache 2.0