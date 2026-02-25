package extension

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/extension/sdk"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestHandleCreateNamespace(t *testing.T) {
	tests := []struct {
		name        string
		args        any
		client      *mockClient
		wantSuccess bool
		checkOutputs func(t *testing.T, result *sdk.OperationResult, ext *Extension)
	}{
		{
			name: "successful create",
			args: map[string]any{
				"prefix": "vm-test",
			},
			client: &mockClient{
				createFn: func(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured, namespace string) (*unstructured.Unstructured, error) {
					return obj.DeepCopy(), nil
				},
			},
			wantSuccess: true,
			checkOutputs: func(t *testing.T, result *sdk.OperationResult, ext *Extension) {
				t.Helper()
				ns, ok := result.Outputs["namespace"]
				if !ok {
					t.Fatal("expected namespace output key")
				}
				if !strings.HasPrefix(ns, "vm-test-") {
					t.Errorf("namespace %q does not have prefix vm-test-", ns)
				}
				// ID should be 8 lowercase alphanumeric chars per mcpchecker#102
				id := strings.TrimPrefix(ns, "vm-test-")
				if len(id) != 8 {
					t.Errorf("expected 8-char id, got %q (len=%d)", id, len(id))
				}

				ext.mu.Lock()
				defer ext.mu.Unlock()
				if len(ext.generatedNamespaces) != 1 {
					t.Fatalf("expected 1 tracked namespace, got %d", len(ext.generatedNamespaces))
				}
				if ext.generatedNamespaces[0] != ns {
					t.Errorf("tracked namespace %q != output namespace %q", ext.generatedNamespaces[0], ns)
				}
			},
		},
		{
			name:        "invalid args type",
			args:        "not a map",
			client:      &mockClient{},
			wantSuccess: false,
		},
		{
			name: "missing prefix",
			args: map[string]any{
				"other": "value",
			},
			client:      &mockClient{},
			wantSuccess: false,
		},
		{
			name: "client error",
			args: map[string]any{
				"prefix": "vm-test",
			},
			client: &mockClient{
				createFn: func(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured, namespace string) (*unstructured.Unstructured, error) {
					return nil, errors.New("connection refused")
				},
			},
			wantSuccess: false,
			checkOutputs: func(t *testing.T, result *sdk.OperationResult, ext *Extension) {
				t.Helper()
				ext.mu.Lock()
				defer ext.mu.Unlock()
				if len(ext.generatedNamespaces) != 0 {
					t.Errorf("expected no tracked namespaces on error, got %d", len(ext.generatedNamespaces))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &Extension{
				Extension: sdk.NewExtension(sdk.ExtensionInfo{Name: "test"}),
				client:    tt.client,
			}

			req := &sdk.OperationRequest{Args: tt.args}
			result, err := ext.handleCreateNamespace(context.Background(), req)

			if err != nil {
				t.Fatalf("handleCreateNamespace() returned error: %v", err)
			}
			if result.Success != tt.wantSuccess {
				t.Errorf("handleCreateNamespace() success = %v, want %v (error: %s)", result.Success, tt.wantSuccess, result.Error)
			}
			if tt.checkOutputs != nil {
				tt.checkOutputs(t, result, ext)
			}
		})
	}
}

func TestHandleDeleteGeneratedNamespaces(t *testing.T) {
	tests := []struct {
		name        string
		tracked     []string
		client      *mockClient
		wantSuccess bool
		checkTracked func(t *testing.T, ext *Extension)
	}{
		{
			name:        "no tracked namespaces",
			tracked:     nil,
			client:      &mockClient{},
			wantSuccess: true,
		},
		{
			name:    "successful deletion of two namespaces",
			tracked: []string{"vm-test-abc123", "vm-test-def456"},
			client: &mockClient{
				deleteFn: func(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string, opts metav1.DeleteOptions) error {
					return nil
				},
			},
			wantSuccess: true,
			checkTracked: func(t *testing.T, ext *Extension) {
				t.Helper()
				ext.mu.Lock()
				defer ext.mu.Unlock()
				if len(ext.generatedNamespaces) != 0 {
					t.Errorf("expected tracking cleared, got %d namespaces", len(ext.generatedNamespaces))
				}
			},
		},
		{
			name:    "not-found error silently ignored",
			tracked: []string{"vm-test-gone"},
			client: &mockClient{
				deleteFn: func(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string, opts metav1.DeleteOptions) error {
					return apierrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, name)
				},
			},
			wantSuccess: true,
		},
		{
			name:    "non-not-found error causes failure",
			tracked: []string{"vm-test-err"},
			client: &mockClient{
				deleteFn: func(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string, opts metav1.DeleteOptions) error {
					return errors.New("permission denied")
				},
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &Extension{
				Extension:           sdk.NewExtension(sdk.ExtensionInfo{Name: "test"}),
				client:              tt.client,
				generatedNamespaces: tt.tracked,
			}

			req := &sdk.OperationRequest{Args: map[string]any{}}
			result, err := ext.handleDeleteGeneratedNamespaces(context.Background(), req)

			if err != nil {
				t.Fatalf("handleDeleteGeneratedNamespaces() returned error: %v", err)
			}
			if result.Success != tt.wantSuccess {
				t.Errorf("handleDeleteGeneratedNamespaces() success = %v, want %v (error: %s)", result.Success, tt.wantSuccess, result.Error)
			}
			if tt.checkTracked != nil {
				tt.checkTracked(t, ext)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	id, err := generateID(8)
	if err != nil {
		t.Fatalf("generateID() error: %v", err)
	}
	if len(id) != 8 {
		t.Errorf("expected 8-char id, got %q (len=%d)", id, len(id))
	}
	for _, c := range id {
		if !strings.ContainsRune(alphanumeric, c) {
			t.Errorf("id contains invalid character %q", string(c))
		}
	}

	// Verify uniqueness (two calls should produce different results)
	id2, err := generateID(8)
	if err != nil {
		t.Fatalf("generateID() error: %v", err)
	}
	if id == id2 {
		t.Errorf("expected unique ids, got %q twice", id)
	}
}
