# kubernetes-extension

A [mcpchecker](https://github.com/mcpchecker/mcpchecker) extension that provides Kubernetes resource operations for task setup, verification, and cleanup.

This extension enables declarative Kubernetes interactions within mcpchecker tasks. It also provides verification-specific operations that don't belong in MCP server tools but are essential for evaluating them. For example, waiting for a pod to reach a specific condition isn't useful as an MCP tool, but is exactly what you need to verify that an MCP server correctly modified a pod.

## Operations

| Operation | Description |
|-----------|-------------|
| `kubernetes.authCanI` | Check if a user or service account can perform an action on a resource |
| `kubernetes.create` | Create a Kubernetes resource |
| `kubernetes.delete` | Delete a Kubernetes resource |
| `kubernetes.getCurrentContext` | Get the current context from kubeconfig |
| `kubernetes.helmInstall` | Install a Helm chart as a release |
| `kubernetes.helmList` | List Helm releases in a namespace or all namespaces |
| `kubernetes.helmUninstall` | Uninstall a Helm release |
| `kubernetes.listContexts` | List all contexts from kubeconfig |
| `kubernetes.viewConfig` | View kubeconfig as YAML (optionally minified) |
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

### Helm Example

Test Helm operations using declarative setup and cleanup:

```yaml
kind: Task
apiVersion: gevals/v1alpha2
metadata:
  name: "list-helm-releases"
  difficulty: easy
spec:
  requires:
    - extension: kubernetes

  setup:
    # Create namespace
    - kubernetes.create:
        apiVersion: v1
        kind: Namespace
        metadata:
          name: helm-test

    # Install a test release
    - kubernetes.helmInstall:
        chart: oci://registry-1.docker.io/bitnamicharts/nginx
        name: test-nginx
        namespace: helm-test

  cleanup:
    # Uninstall the release
    - kubernetes.helmUninstall:
        name: test-nginx
        namespace: helm-test

    # Delete namespace
    - kubernetes.delete:
        apiVersion: v1
        kind: Namespace
        metadata:
          name: helm-test
        ignoreNotFound: true

  prompt:
    inline: List all Helm releases in the cluster
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

### kubernetes.helmInstall

Installs a Helm chart as a release. Supports chart repositories and OCI registries.

```yaml
- kubernetes.helmInstall:
    chart: oci://registry-1.docker.io/bitnamicharts/nginx
    name: my-nginx        # optional, generates name if not provided
    namespace: default    # optional
    values:               # optional Helm values
      replicaCount: 2
      service:
        type: LoadBalancer
```

### kubernetes.helmList

Lists Helm releases in a namespace or across all namespaces.

```yaml
# List in specific namespace
- kubernetes.helmList:
    namespace: default

# List across all namespaces
- kubernetes.helmList:
    allNamespaces: true
```

**Outputs:**
- `releases`: Information about found Helm releases (name, namespace, status, chart)

### kubernetes.helmUninstall

Uninstalls a Helm release. Gracefully handles releases that don't exist.

```yaml
- kubernetes.helmUninstall:
    name: my-nginx
    namespace: default    # optional
```

### kubernetes.listContexts

Lists all contexts from the kubeconfig file, including which one is currently active.

```yaml
- kubernetes.listContexts:
    # No parameters required
```

**Outputs:**
- `current`: Name of the current context
- `count`: Number of contexts found

### kubernetes.getCurrentContext

Returns the current context name from the kubeconfig.

```yaml
- kubernetes.getCurrentContext:
    # No parameters required
```

**Outputs:**
- `context`: Name of the current context

### kubernetes.viewConfig

Views the kubeconfig as YAML, optionally minified to show only the current context.

```yaml
- kubernetes.viewConfig:
    minify: false  # optional, defaults to false
```

**Outputs:**
- `config`: The kubeconfig content as YAML

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, project structure, and guidelines for adding new operations.

## License

Apache-2.0
