package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosf/k8s-pod-manager/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// ClientInterface defines the interface for Kubernetes operations
// This enables dependency injection and testing
type ClientInterface interface {
	kubernetes.Interface
}

var Client *kubernetes.Clientset

func InitClient() error {
	var cfg *rest.Config
	var err error

	if config.AppConfig.InCluster {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		kubeconfig := config.AppConfig.KubeConfig
		if kubeconfig == "" {
			if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
				kubeconfig = envConfig
			} else if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			} else {
				return fmt.Errorf("kubeconfig not found")
			}
		}

		if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
			return fmt.Errorf("kubeconfig file does not exist: %s", kubeconfig)
		}

		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	Client, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return nil
}