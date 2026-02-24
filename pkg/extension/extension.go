package extension

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mcpchecker/mcpchecker/pkg/extension/sdk"
	"k8s.io/client-go/dynamic"
	authorizationv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// Extension wraps the SDK extension with Kubernetes client
type Extension struct {
	*sdk.Extension
	client ResourceClient

	mu                   sync.Mutex
	generatedNamespaces []string
}

// New creates a new Kubernetes extension
func New() *Extension {
	ext := &Extension{}
	ext.Extension = sdk.NewExtension(
		sdk.ExtensionInfo{
			Name:        "kubernetes",
			Version:     "0.1.0",
			Description: "Kubernetes resource operations using client-go",
		},
		sdk.WithInitializeHandler(ext.handleInitialize),
	)

	ext.registerOperations()
	return ext
}

func (e *Extension) handleInitialize(config map[string]any) error {
	kubeconfigPath := ""

	if path, ok := config["kubeconfig"].(string); ok {
		kubeconfigPath = path
	}

	// Expand ~ to home directory
	if strings.HasPrefix(kubeconfigPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, kubeconfigPath[1:])
	}

	// If no kubeconfig specified, use default
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	// Validate kubeconfig file exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig not found: %s", kubeconfigPath)
	}

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig from %s: %w", kubeconfigPath, err)
	}

	client, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	authzClient, err := authorizationv1client.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create authorization client: %w", err)
	}

	e.client = &dynamicClientAdapter{
		client:         client,
		authzClient:    authzClient,
		kubeconfigPath: kubeconfigPath,
	}
	return nil
}

// Run starts the extension, listening for JSON-RPC messages on stdin/stdout
func (e *Extension) Run(ctx context.Context) error {
	return e.Extension.Run(ctx)
}
