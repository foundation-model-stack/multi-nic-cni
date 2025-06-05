package plugin_test

import (
	"context"
	"fmt"

	. "github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// Mock resource interface
type mockResourceInterface struct {
	items     []map[string]interface{}
	namespace string
}

// List implements dynamic.ResourceInterface
func (m *mockResourceInterface) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	if len(m.items) == 0 {
		return &unstructured.UnstructuredList{}, nil
	}

	list := &unstructured.UnstructuredList{
		Items: make([]unstructured.Unstructured, len(m.items)),
	}

	for i, item := range m.items {
		list.Items[i] = unstructured.Unstructured{
			Object: item,
		}
	}
	return list, nil
}

// Get implements dynamic.ResourceInterface
func (m *mockResourceInterface) Get(ctx context.Context, name string, opts metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}

// Create implements dynamic.ResourceInterface
func (m *mockResourceInterface) Create(ctx context.Context, obj *unstructured.Unstructured, opts metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}

// Update implements dynamic.ResourceInterface
func (m *mockResourceInterface) Update(ctx context.Context, obj *unstructured.Unstructured, opts metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}

// UpdateStatus implements dynamic.ResourceInterface
func (m *mockResourceInterface) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, opts metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}

// Delete implements dynamic.ResourceInterface
func (m *mockResourceInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions, subresources ...string) error {
	return fmt.Errorf("not implemented")
}

// DeleteCollection implements dynamic.ResourceInterface
func (m *mockResourceInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return fmt.Errorf("not implemented")
}

// Watch implements dynamic.ResourceInterface
func (m *mockResourceInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("not implemented")
}

// Patch implements dynamic.ResourceInterface
func (m *mockResourceInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}

// Apply implements dynamic.ResourceInterface
func (m *mockResourceInterface) Apply(ctx context.Context, name string, obj *unstructured.Unstructured, opts metav1.ApplyOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}

// ApplyStatus implements dynamic.ResourceInterface
func (m *mockResourceInterface) ApplyStatus(ctx context.Context, name string, obj *unstructured.Unstructured, opts metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}

// Namespace implements dynamic.ResourceInterface
func (m *mockResourceInterface) Namespace(namespace string) dynamic.ResourceInterface {
	m.namespace = namespace
	return m
}

type mockDynamicClient struct {
	resources map[string]*mockResourceInterface
}

func (m *mockDynamicClient) Resource(gvr schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	if ri, exists := m.resources["default"]; exists {
		return ri
	}
	return &mockResourceInterface{items: []map[string]interface{}{}}
}

var _ = Describe("DynamicHandler", func() {
	var (
		handler *DynamicHandler
		client  *mockDynamicClient
		gvr     schema.GroupVersionResource
	)

	BeforeEach(func() {
		client = &mockDynamicClient{
			resources: make(map[string]*mockResourceInterface),
		}

		gvr = schema.GroupVersionResource{
			Group:    "test.group",
			Version:  "v1",
			Resource: "testresources",
		}

		handler = &DynamicHandler{
			DYN: client,
			GVR: gvr,
		}
	})

	Context("GetFirst", func() {
		It("should return the first item when items exist", func() {
			client.resources["default"] = &mockResourceInterface{
				items: []map[string]interface{}{
					{
						"apiVersion": "test.group/v1",
						"kind":       "TestResource",
						"metadata": map[string]interface{}{
							"name":      "test1",
							"namespace": "default",
						},
						"spec": map[string]interface{}{
							"key": "value1",
						},
					},
					{
						"apiVersion": "test.group/v1",
						"kind":       "TestResource",
						"metadata": map[string]interface{}{
							"name":      "test2",
							"namespace": "default",
						},
						"spec": map[string]interface{}{
							"key": "value2",
						},
					},
				},
			}

			var result map[string]interface{}
			err := handler.GetFirst("default", &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result["metadata"].(map[string]interface{})["name"]).To(Equal("test1"))
		})

		It("should return error when no items exist", func() {
			client.resources["default"] = &mockResourceInterface{
				items: []map[string]interface{}{},
			}

			var result map[string]interface{}
			err := handler.GetFirst("default", &result)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no item"))
		})

		It("should handle unmarshalling errors", func() {
			client.resources["default"] = &mockResourceInterface{
				items: []map[string]interface{}{
					{
						"metadata": map[string]interface{}{
							"invalid": make(chan int),
						},
					},
				},
			}

			var result map[string]interface{}
			err := handler.GetFirst("default", &result)
			Expect(err).To(HaveOccurred())
		})
	})
})
