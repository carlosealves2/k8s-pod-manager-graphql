package services

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// NewTestKubernetesClient creates a new test client that implements kubernetes.Interface
func NewTestKubernetesClient() kubernetes.Interface {
	return fake.NewSimpleClientset()
}

// NewTestKubernetesClientWithObjects creates a new test client with initial objects
func NewTestKubernetesClientWithObjects(objects ...runtime.Object) kubernetes.Interface {
	return fake.NewSimpleClientset(objects...)
}