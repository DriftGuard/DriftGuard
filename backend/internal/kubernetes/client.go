package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	config    *config.KubernetesConfig
	clientset *kubernetes.Clientset
	dynamic   dynamic.Interface
	mapper    meta.RESTMapper
	logger    *zap.Logger
}

func NewClient(cfg *config.KubernetesConfig, logger *zap.Logger) (*Client, error) {
	var kubeConfig *rest.Config
	var err error

	if cfg.ConfigPath != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	} else {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	}

	if cfg.Context != "" {
		configAccess := clientcmd.NewDefaultPathOptions()
		config, err := configAccess.GetStartingConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get starting config: %w", err)
		}

		context, exists := config.Contexts[cfg.Context]
		if !exists {
			return nil, fmt.Errorf("context %s not found", cfg.Context)
		}

		_, exists = config.Clusters[context.Cluster]
		if !exists {
			return nil, fmt.Errorf("cluster %s not found", context.Cluster)
		}

		_, exists = config.AuthInfos[context.AuthInfo]
		if !exists {
			return nil, fmt.Errorf("auth info %s not found", context.AuthInfo)
		}

		kubeConfig, err = clientcmd.NewNonInteractiveClientConfig(*config, cfg.Context, &clientcmd.ConfigOverrides{}, configAccess).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create client config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient := clientset.Discovery()
	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get API group resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	return &Client{
		config:    cfg,
		clientset: clientset,
		dynamic:   dynamicClient,
		mapper:    mapper,
		logger:    logger,
	}, nil
}

func (c *Client) GetResource(kind, namespace, name string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c.logger.Debug("Fetching resource from Kubernetes",
		zap.String("kind", kind),
		zap.String("namespace", namespace),
		zap.String("name", name))

	gvr, err := c.getGVR(kind)
	if err != nil {
		return nil, fmt.Errorf("failed to get GVR for kind %s: %w", kind, err)
	}

	var resource *unstructured.Unstructured
	if namespace == "" {
		resource, err = c.dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		resource, err = c.dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s/%s/%s: %w", kind, namespace, name, err)
	}

	result := resource.UnstructuredContent()
	c.logger.Debug("Successfully fetched resource", zap.String("resource_id", fmt.Sprintf("%s:%s:%s", kind, namespace, name)))

	return result, nil
}

func (c *Client) ListResources(kind, namespace string) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c.logger.Debug("Listing resources from Kubernetes",
		zap.String("kind", kind),
		zap.String("namespace", namespace))

	gvr, err := c.getGVR(kind)
	if err != nil {
		return nil, fmt.Errorf("failed to get GVR for kind %s: %w", kind, err)
	}

	var list *unstructured.UnstructuredList
	if namespace == "" {
		list, err = c.dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		list, err = c.dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list resources of kind %s in namespace %s: %w", kind, namespace, err)
	}

	var results []map[string]interface{}
	for _, item := range list.Items {
		results = append(results, item.UnstructuredContent())
	}

	c.logger.Debug("Successfully listed resources",
		zap.String("kind", kind),
		zap.String("namespace", namespace),
		zap.Int("count", len(results)))

	return results, nil
}

func (c *Client) GetNamespaces() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if len(c.config.Namespaces) > 0 {
		return c.config.Namespaces, nil
	}

	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var result []string
	for _, ns := range namespaces.Items {
		result = append(result, ns.Name)
	}

	return result, nil
}

func (c *Client) getGVR(kind string) (schema.GroupVersionResource, error) {
	resourceMappings := map[string]schema.GroupVersionResource{
		"Pod":                   {Group: "", Version: "v1", Resource: "pods"},
		"Deployment":            {Group: "apps", Version: "v1", Resource: "deployments"},
		"Service":               {Group: "", Version: "v1", Resource: "services"},
		"ConfigMap":             {Group: "", Version: "v1", Resource: "configmaps"},
		"Secret":                {Group: "", Version: "v1", Resource: "secrets"},
		"PersistentVolumeClaim": {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"StatefulSet":           {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"DaemonSet":             {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"Ingress":               {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"Job":                   {Group: "batch", Version: "v1", Resource: "jobs"},
		"CronJob":               {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"Namespace":             {Group: "", Version: "v1", Resource: "namespaces"},
		"Node":                  {Group: "", Version: "v1", Resource: "nodes"},
		"PersistentVolume":      {Group: "", Version: "v1", Resource: "persistentvolumes"},
		"StorageClass":          {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
		"Role":                  {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
		"RoleBinding":           {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		"ClusterRole":           {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
		"ClusterRoleBinding":    {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
	}

	if gvr, exists := resourceMappings[kind]; exists {
		return gvr, nil
	}

	gvk := schema.GroupVersionKind{Kind: kind}
	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to get REST mapping for kind %s: %w", kind, err)
	}

	return mapping.Resource, nil
}

func (c *Client) isSystemNamespace(name string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
	}

	for _, sysNS := range systemNamespaces {
		if name == sysNS {
			return true
		}
	}

	return false
}

func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	return err
}
