package extension

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/mcpchecker/mcpchecker/pkg/extension/sdk"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var namespaceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}

const alphanumeric = "abcdefghijklmnopqrstuvwxyz0123456789"

// generateID returns a random lowercase alphanumeric string of the given length.
// This matches the {random.id} spec from mcpchecker/mcpchecker#102.
func generateID(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random id: %w", err)
	}
	for i := range b {
		b[i] = alphanumeric[int(b[i])%len(alphanumeric)]
	}
	return string(b), nil
}

func (e *Extension) handleCreateNamespace(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
	if e.client == nil {
		return sdk.Failure(fmt.Errorf("kubernetes client not initialized")), nil
	}

	args, ok := req.Args.(map[string]any)
	if !ok {
		return sdk.Failure(fmt.Errorf("args must be an object")), nil
	}

	prefix, _ := args["prefix"].(string)
	if prefix == "" {
		return sdk.Failure(fmt.Errorf("prefix is required")), nil
	}

	id, err := generateID(8)
	if err != nil {
		return sdk.Failure(err), nil
	}

	name := fmt.Sprintf("%s-%s", prefix, id)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]any{
				"name": name,
			},
		},
	}

	e.LogInfo(ctx, "Creating namespace", map[string]any{
		"name": name,
	})

	result, err := e.client.Create(ctx, namespaceGVR, obj, "")
	if err != nil {
		e.LogError(ctx, "Failed to create namespace", map[string]any{
			"name":  name,
			"error": err.Error(),
		})
		return sdk.Failure(fmt.Errorf("failed to create namespace: %w", err)), nil
	}

	e.mu.Lock()
	e.generatedNamespaces = append(e.generatedNamespaces, result.GetName())
	e.mu.Unlock()

	e.LogInfo(ctx, "Namespace created successfully", map[string]any{
		"name": result.GetName(),
	})

	return sdk.SuccessWithOutputs(
		fmt.Sprintf("Created namespace %s", result.GetName()),
		map[string]string{
			"namespace": result.GetName(),
		},
	), nil
}

func (e *Extension) handleDeleteGeneratedNamespaces(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
	if e.client == nil {
		return sdk.Failure(fmt.Errorf("kubernetes client not initialized")), nil
	}

	e.mu.Lock()
	namespaces := make([]string, len(e.generatedNamespaces))
	copy(namespaces, e.generatedNamespaces)
	e.generatedNamespaces = nil
	e.mu.Unlock()

	if len(namespaces) == 0 {
		return sdk.Success("No generated namespaces to delete"), nil
	}

	e.LogInfo(ctx, "Deleting generated namespaces", map[string]any{
		"count":      len(namespaces),
		"namespaces": namespaces,
	})

	propagation := metav1.DeletePropagationForeground
	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}

	var errs []string
	for _, ns := range namespaces {
		err := e.client.Delete(ctx, namespaceGVR, ns, "", deleteOpts)
		if err != nil {
			if apierrors.IsNotFound(err) {
				e.LogInfo(ctx, "Namespace already deleted (ignored)", map[string]any{
					"name": ns,
				})
				continue
			}
			e.LogError(ctx, "Failed to delete namespace", map[string]any{
				"name":  ns,
				"error": err.Error(),
			})
			errs = append(errs, fmt.Sprintf("%s: %s", ns, err.Error()))
		}
	}

	if len(errs) > 0 {
		return sdk.Failure(fmt.Errorf("failed to delete namespaces: %s", strings.Join(errs, "; "))), nil
	}

	return sdk.Success(fmt.Sprintf("Deleted %d generated namespace(s)", len(namespaces))), nil
}
