# kubernetes-extension

A [mcpchecker](https://github.com/mcpchecker/mcpchecker) extension that provides Kubernetes resource operations for task setup, verification, and cleanup.

This extension enables declarative Kubernetes interactions within mcpchecker tasks. It also provides verification-specific operations that don't belong in MCP server tools but are essential for evaluating them. For example, waiting for a pod to reach a specific condition isn't useful as an MCP tool, but is exactly what you need to verify that an MCP server correctly modified a pod.

## Operations

| Operation | Description |
|-----------|-------------|
| `kubernetes.authCanI` | Check if a user or service account can perform an action on a resource |
| `kubernetes.create` | Create a Kubernetes resource |
| `kubernetes.delete` | Delete a Kubernetes resource |
| `kubernetes.wait` | Wait for a condition on a resource (e.g., `Ready`, `Available`) |

## Configuration

Add the extension to your `eval.yaml`:

```yaml
kind: Eval
metadata:
  name: "kubernetes-basic-operations"
config:
  agent:
    type: "file"
    path: agent.yaml
  mcpConfigFile: mcp-config.yaml
  extensions:
    kubernetes:
      package: https://github.com/mcpchecker/kubernetes-extension@v0.0.2
      config:
        kubeconfig: ~/.kube/config  # optional, defaults to ~/.kube/config
  taskSets:
    - glob: tasks/*/*.yaml
```

## Task Usage

Declare the extension requirement and use operations in `setup`, `verify`, and `cleanup` phases:

```yaml
kind: Task
apiVersion: gevals/v1alpha2
metadata:
  name: "create-nginx-pod"
  difficulty: easy
spec:
  requires:
    - extension: kubernetes

  setup:
    - kubernetes.delete:
        apiVersion: v1
        kind: Namespace
        metadata:
          name: test-namespace
        ignoreNotFound: true
    - kubernetes.create:
        apiVersion: v1
        kind: Namespace
        metadata:
          name: test-namespace

  verify:
    - kubernetes.wait:
        apiVersion: v1
        kind: Pod
        metadata:
          name: web-server
          namespace: test-namespace
        condition: Ready
        timeout: 120s

  cleanup:
    - kubernetes.delete:
        apiVersion: v1
        kind: Namespace
        metadata:
          name: test-namespace
        ignoreNotFound: true

  prompt:
    inline: Create an nginx pod named web-server in the test-namespace namespace
```

## Operation Reference

### kubernetes.create

Creates a Kubernetes resource using standard manifest fields.

```yaml
- kubernetes.create:
    apiVersion: v1
    kind: Pod
    metadata:
      name: my-pod
      namespace: default
    spec:
      containers:
        - name: nginx
          image: nginx:latest
```

### kubernetes.delete

Deletes a Kubernetes resource. Use `ignoreNotFound: true` to skip errors when the resource doesn't exist.

```yaml
- kubernetes.delete:
    apiVersion: v1
    kind: Namespace
    metadata:
      name: my-namespace
    ignoreNotFound: true
```

### kubernetes.wait

Waits for a condition on a resource. Supports configurable timeout and expected status.

```yaml
- kubernetes.wait:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: my-deployment
      namespace: default
    condition: Available
    status: "True"    # optional, defaults to "True"
    timeout: 5m       # optional, defaults to 60s
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, project structure, and guidelines for adding new operations.

## License

Apache-2.0
