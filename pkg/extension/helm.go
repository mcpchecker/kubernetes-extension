package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mcpchecker/mcpchecker/pkg/extension/sdk"
)

// handleHelmInstall installs a Helm chart as a release
func (e *Extension) handleHelmInstall(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
	args, ok := req.Args.(map[string]any)
	if !ok {
		return sdk.Failure(fmt.Errorf("args must be an object")), nil
	}

	chart, ok := args["chart"].(string)
	if !ok || chart == "" {
		return sdk.Failure(fmt.Errorf("chart parameter is required")), nil
	}

	// Optional parameters
	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)
	values, _ := args["values"].(map[string]interface{})

	cmdArgs := []string{"install"}

	if name != "" {
		cmdArgs = append(cmdArgs, name)
	} else {
		cmdArgs = append(cmdArgs, "--generate-name")
	}

	cmdArgs = append(cmdArgs, chart)

	if namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace", namespace)
	}

	if e.kubeconfigPath != "" {
		cmdArgs = append(cmdArgs, "--kubeconfig", e.kubeconfigPath)
	}

	// Add values as --set flags
	for k, v := range values {
		cmdArgs = append(cmdArgs, "--set", fmt.Sprintf("%s=%v", k, v))
	}

	e.LogInfo(ctx, "Installing Helm chart", map[string]any{
		"chart":     chart,
		"name":      name,
		"namespace": namespace,
	})

	cmd := exec.CommandContext(ctx, "helm", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.LogError(ctx, "Helm install failed", map[string]any{
			"chart": chart,
			"error": err.Error(),
		})
		return sdk.Failure(fmt.Errorf("helm install failed: %s\nOutput: %s", err, string(output))), nil
	}

	e.LogInfo(ctx, "Helm chart installed successfully", map[string]any{
		"chart": chart,
		"name":  name,
	})

	return sdk.Success(fmt.Sprintf("Helm chart installed successfully\n%s", string(output))), nil
}

// handleHelmList lists Helm releases
func (e *Extension) handleHelmList(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
	args, ok := req.Args.(map[string]any)
	if !ok {
		return sdk.Failure(fmt.Errorf("args must be an object")), nil
	}

	cmdArgs := []string{"list", "--output", "json"}

	namespace, _ := args["namespace"].(string)
	allNamespaces, _ := args["allNamespaces"].(bool)

	if allNamespaces {
		cmdArgs = append(cmdArgs, "--all-namespaces")
	} else if namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace", namespace)
	}

	if e.kubeconfigPath != "" {
		cmdArgs = append(cmdArgs, "--kubeconfig", e.kubeconfigPath)
	}

	e.LogInfo(ctx, "Listing Helm releases", map[string]any{
		"namespace":      namespace,
		"allNamespaces":  allNamespaces,
	})

	cmd := exec.CommandContext(ctx, "helm", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.LogError(ctx, "Helm list failed", map[string]any{
			"error": err.Error(),
		})
		return sdk.Failure(fmt.Errorf("helm list failed: %s\nOutput: %s", err, string(output))), nil
	}

	// Parse JSON output
	var releases []map[string]interface{}
	if len(output) > 0 {
		if err := json.Unmarshal(output, &releases); err != nil {
			return sdk.Failure(fmt.Errorf("failed to parse helm list output: %s", err)), nil
		}
	}

	if len(releases) == 0 {
		return sdk.Success("No Helm releases found"), nil
	}

	// Format releases as a readable string
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d Helm release(s):\n", len(releases)))
	for _, release := range releases {
		name, _ := release["name"].(string)
		ns, _ := release["namespace"].(string)
		status, _ := release["status"].(string)
		chart, _ := release["chart"].(string)
		result.WriteString(fmt.Sprintf("  - %s (namespace: %s, status: %s, chart: %s)\n", name, ns, status, chart))
	}

	return sdk.Success(result.String()), nil
}

// handleHelmUninstall uninstalls a Helm release
func (e *Extension) handleHelmUninstall(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
	args, ok := req.Args.(map[string]any)
	if !ok {
		return sdk.Failure(fmt.Errorf("args must be an object")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return sdk.Failure(fmt.Errorf("name parameter is required")), nil
	}

	cmdArgs := []string{"uninstall", name}

	namespace, _ := args["namespace"].(string)
	if namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace", namespace)
	}

	if e.kubeconfigPath != "" {
		cmdArgs = append(cmdArgs, "--kubeconfig", e.kubeconfigPath)
	}

	e.LogInfo(ctx, "Uninstalling Helm release", map[string]any{
		"name":      name,
		"namespace": namespace,
	})

	cmd := exec.CommandContext(ctx, "helm", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(string(output), "not found") {
			e.LogInfo(ctx, "Helm release not found (ignored)", map[string]any{
				"name": name,
			})
			return sdk.Success(fmt.Sprintf("Helm release '%s' not found (already uninstalled)", name)), nil
		}
		e.LogError(ctx, "Helm uninstall failed", map[string]any{
			"name":  name,
			"error": err.Error(),
		})
		return sdk.Failure(fmt.Errorf("helm uninstall failed: %s\nOutput: %s", err, string(output))), nil
	}

	e.LogInfo(ctx, "Helm release uninstalled successfully", map[string]any{
		"name": name,
	})

	return sdk.Success(fmt.Sprintf("Helm release '%s' uninstalled successfully\n%s", name, string(output))), nil
}
