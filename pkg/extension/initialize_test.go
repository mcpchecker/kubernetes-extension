package extension

import (
	"os"
	"path/filepath"
	"testing"
)

const testKubeconfig = `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
users:
- name: test
  user:
    token: test
`

func TestHandleInitializeUsesKubeconfigEnvWhenConfigUnset(t *testing.T) {
	homeDir := t.TempDir()
	defaultPath := filepath.Join(homeDir, ".kube", "config")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte(testKubeconfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default kubeconfig) error = %v", err)
	}

	envPath := filepath.Join(t.TempDir(), "env-config")
	if err := os.WriteFile(envPath, []byte(testKubeconfig), 0o644); err != nil {
		t.Fatalf("WriteFile(env kubeconfig) error = %v", err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("KUBECONFIG", envPath)

	ext := &Extension{}
	if err := ext.handleInitialize(map[string]any{}); err != nil {
		t.Fatalf("handleInitialize() error = %v", err)
	}

	if ext.kubeconfigPath != envPath {
		t.Fatalf("kubeconfigPath = %q, want %q", ext.kubeconfigPath, envPath)
	}
}

func TestHandleInitializePrefersExplicitConfigOverEnv(t *testing.T) {
	explicitPath := filepath.Join(t.TempDir(), "explicit-config")
	if err := os.WriteFile(explicitPath, []byte(testKubeconfig), 0o644); err != nil {
		t.Fatalf("WriteFile(explicit kubeconfig) error = %v", err)
	}

	envPath := filepath.Join(t.TempDir(), "env-config")
	if err := os.WriteFile(envPath, []byte(testKubeconfig), 0o644); err != nil {
		t.Fatalf("WriteFile(env kubeconfig) error = %v", err)
	}

	t.Setenv("KUBECONFIG", envPath)

	ext := &Extension{}
	if err := ext.handleInitialize(map[string]any{"kubeconfig": explicitPath}); err != nil {
		t.Fatalf("handleInitialize() error = %v", err)
	}

	if ext.kubeconfigPath != explicitPath {
		t.Fatalf("kubeconfigPath = %q, want %q", ext.kubeconfigPath, explicitPath)
	}
}

func TestHandleInitializeFallsBackToHomeKubeconfig(t *testing.T) {
	homeDir := t.TempDir()
	defaultPath := filepath.Join(homeDir, ".kube", "config")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte(testKubeconfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default kubeconfig) error = %v", err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("KUBECONFIG", "")

	ext := &Extension{}
	if err := ext.handleInitialize(map[string]any{}); err != nil {
		t.Fatalf("handleInitialize() error = %v", err)
	}

	if ext.kubeconfigPath != defaultPath {
		t.Fatalf("kubeconfigPath = %q, want %q", ext.kubeconfigPath, defaultPath)
	}
}
