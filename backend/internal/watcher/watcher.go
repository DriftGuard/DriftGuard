package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"github.com/DriftGuard/core/internal/database"
	"github.com/DriftGuard/core/pkg/models"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// TODO: Kubernetes Watcher Implementation
//
// PHASE 2 PRIORITY 2: Implement real Kubernetes API integration
//
// Current Status: Mock implementation - returns empty struct
// Next Steps:
// 1. Add Kubernetes client dependencies: go get k8s.io/client-go
// 2. Implement real Kubernetes client with proper authentication
// 3. Set up resource watchers for configured resources
// 4. Implement event processing and filtering
// 5. Add resource comparison logic
// 6. Implement retry and error handling
// 7. Add metrics and monitoring for watcher health
// 8. Create resource caching for performance
//
// Required Methods to Implement:
// - WatchResources(ctx context.Context) (<-chan *models.KubernetesResource, error)
// - GetResource(namespace, name, kind string) (*models.KubernetesResource, error)
// - ListResources(namespace, kind string) ([]*models.KubernetesResource, error)
// - GetResourceVersion(namespace, name, kind string) (string, error)
// - WatchResourceChanges(namespace, kind string) (<-chan *ResourceChange, error)
// - ValidateResourceAccess(namespace, kind string) error
//
// Kubernetes Resources to Monitor:
// - Deployments
// - Services
// - ConfigMaps
// - Secrets
// - Ingress
// - PersistentVolumeClaims
// - ServiceAccounts
// - NetworkPolicies

type KubernetesWatcher struct {
	clientset *kubernetes.Clientset
	config    config.KubernetesConfig
	logger    *zap.Logger
	factory   informers.SharedInformerFactory
	stopCh    chan struct{}
	db        *database.Database

	// Efficiency controls
	snapshotInterval time.Duration
	lastSnapshot     map[string]time.Time
	snapshotMutex    sync.RWMutex
	enableSnapshots  bool
}

func NewKubernetesWatcher(cfg config.KubernetesConfig, logger *zap.Logger, db *database.Database) (*KubernetesWatcher, error) {
	var config *rest.Config
	var err error
	if cfg.ConfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", cfg.ConfigPath)
		if err != nil {
			logger.Error("Failed to load kubeconfig", zap.Error(err))
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			logger.Error("Failed to load in-cluster config", zap.Error(err))
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error("Failed to create Kubernetes clientset", zap.Error(err))
		return nil, err
	}

	// Create namespace-specific informer factory
	var factory informers.SharedInformerFactory
	if len(cfg.Namespaces) > 0 {
		// Use namespace-specific informers
		factory = informers.NewSharedInformerFactoryWithOptions(clientset, 30*time.Second,
			informers.WithNamespace(cfg.Namespaces[0])) // For now, use the first namespace
		logger.Info("Created namespace-specific informer", zap.String("namespace", cfg.Namespaces[0]))
	} else {
		// Use cluster-wide informers
		factory = informers.NewSharedInformerFactory(clientset, 30*time.Second)
		logger.Info("Created cluster-wide informer")
	}

	// Set default snapshot interval if not configured
	snapshotInterval := cfg.SnapshotInterval
	if snapshotInterval == 0 {
		snapshotInterval = 5 * time.Minute
	}

	return &KubernetesWatcher{
		clientset:        clientset,
		config:           cfg,
		logger:           logger,
		factory:          factory,
		stopCh:           make(chan struct{}),
		db:               db,
		snapshotInterval: snapshotInterval,
		lastSnapshot:     make(map[string]time.Time),
		enableSnapshots:  cfg.EnableSnapshots,
	}, nil
}

type ResourceEventType string

const (
	ResourceAdded   ResourceEventType = "added"
	ResourceUpdated ResourceEventType = "updated"
	ResourceDeleted ResourceEventType = "deleted"
)

type ResourceEvent struct {
	Type     ResourceEventType
	Resource *models.KubernetesResource
}

// WatchResources streams resource events to a channel for consumption by other components
func (w *KubernetesWatcher) WatchResources(ctx context.Context) error {
	// Create a map of configured resources for quick lookup
	configuredResources := make(map[string]bool)
	for _, resource := range w.config.Resources {
		configuredResources[strings.ToLower(resource)] = true
	}

	w.logger.Info("Starting resource watchers",
		zap.Strings("configured_resources", w.config.Resources),
		zap.Strings("namespaces", w.config.Namespaces))

	// Only watch resources that are explicitly configured
	if configuredResources["deployments"] {
		w.logger.Info("Setting up Deployment watcher")
		deploymentInformer := w.factory.Apps().V1().Deployments().Informer()
		deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("Deployment added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					// Only save if this is a significant change
					if w.shouldSaveSnapshot(resource) {
						snapshot := &models.ConfigurationSnapshot{
							Namespace:    resource.Metadata.Namespace,
							ResourceType: resource.Kind,
							ResourceName: resource.Metadata.Name,
							LiveState:    marshalToJSON(resource),
							CreatedAt:    time.Now(),
							UpdatedAt:    time.Now(),
						}
						if err := w.db.SaveSnapshot(snapshot); err != nil {
							w.logger.Error("Failed to save snapshot", zap.Error(err))
						}
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("Deployment updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("Deployment deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	// Watch ConfigMaps only if configured
	if configuredResources["configmaps"] {
		w.logger.Info("Setting up ConfigMap watcher")
		configMapInformer := w.factory.Core().V1().ConfigMaps().Informer()
		configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("ConfigMap added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("ConfigMap updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("ConfigMap deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	// Watch Services only if configured
	if configuredResources["services"] {
		w.logger.Info("Setting up Service watcher")
		serviceInformer := w.factory.Core().V1().Services().Informer()
		serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("Service added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("Service updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("Service deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	// Watch Pods only if configured
	if configuredResources["pods"] {
		w.logger.Info("Setting up Pod watcher")
		podInformer := w.factory.Core().V1().Pods().Informer()
		podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("Pod added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("Pod updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("Pod deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	// Watch Secrets only if configured
	if configuredResources["secrets"] {
		w.logger.Info("Setting up Secret watcher")
		secretInformer := w.factory.Core().V1().Secrets().Informer()
		secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("Secret added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("Secret updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("Secret deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	// Watch Ingress only if configured
	if configuredResources["ingresses"] {
		w.logger.Info("Setting up Ingress watcher")
		ingressInformer := w.factory.Networking().V1().Ingresses().Informer()
		ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("Ingress added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("Ingress updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("Ingress deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	// Watch PersistentVolumeClaims only if configured
	if configuredResources["persistentvolumeclaims"] {
		w.logger.Info("Setting up PersistentVolumeClaim watcher")
		pvcInformer := w.factory.Core().V1().PersistentVolumeClaims().Informer()
		pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("PersistentVolumeClaim added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("PersistentVolumeClaim updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("PersistentVolumeClaim deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	// Watch ServiceAccounts only if configured
	if configuredResources["serviceaccounts"] {
		w.logger.Info("Setting up ServiceAccount watcher")
		saInformer := w.factory.Core().V1().ServiceAccounts().Informer()
		saInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("ServiceAccount added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("ServiceAccount updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("ServiceAccount deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	// Watch NetworkPolicies only if configured
	if configuredResources["networkpolicies"] {
		w.logger.Info("Setting up NetworkPolicy watcher")
		npInformer := w.factory.Networking().V1().NetworkPolicies().Informer()
		npInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.logger.Info("NetworkPolicy added", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.logger.Info("NetworkPolicy updated", zap.Any("new_object", newObj))
				resource := toKubernetesResource(newObj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				w.logger.Info("NetworkPolicy deleted", zap.Any("object", obj))
				resource := toKubernetesResource(obj)
				if resource != nil {
					snapshot := &models.ConfigurationSnapshot{
						Namespace:    resource.Metadata.Namespace,
						ResourceType: resource.Kind,
						ResourceName: resource.Metadata.Name,
						LiveState:    marshalToJSON(resource),
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					if err := w.db.SaveSnapshot(snapshot); err != nil {
						w.logger.Error("Failed to save snapshot", zap.Error(err))
					}
				}
			},
		})
	}

	w.factory.Start(w.stopCh)
	w.logger.Info("Started informers for configured resources",
		zap.Strings("monitored_resources", w.config.Resources))
	<-ctx.Done()
	close(w.stopCh)
	w.logger.Info("Stopped all informers")
	return nil
}

// isResourceConfigured checks if a resource type is configured for monitoring
func (w *KubernetesWatcher) isResourceConfigured(resourceType string) bool {
	for _, configured := range w.config.Resources {
		if strings.EqualFold(configured, resourceType) {
			return true
		}
	}
	return false
}

// GetResource fetches a resource from the informer cache (only for configured resources)
func (w *KubernetesWatcher) GetResource(namespace, name, kind string) (*models.KubernetesResource, error) {
	// Check if this resource type is configured for monitoring
	if !w.isResourceConfigured(kind) {
		return nil, fmt.Errorf("resource type '%s' is not configured for monitoring", kind)
	}

	switch strings.ToLower(kind) {
	case "deployment":
		inf := w.factory.Apps().V1().Deployments().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	case "service":
		inf := w.factory.Core().V1().Services().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	case "configmap":
		inf := w.factory.Core().V1().ConfigMaps().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	case "pod":
		inf := w.factory.Core().V1().Pods().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	case "secret":
		inf := w.factory.Core().V1().Secrets().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	case "ingress":
		inf := w.factory.Networking().V1().Ingresses().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	case "persistentvolumeclaim":
		inf := w.factory.Core().V1().PersistentVolumeClaims().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	case "serviceaccount":
		inf := w.factory.Core().V1().ServiceAccounts().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	case "networkpolicy":
		inf := w.factory.Networking().V1().NetworkPolicies().Informer()
		_, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return nil, err
		}
		return &models.KubernetesResource{}, nil // TODO: Convert obj
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", kind)
	}
}

// ListResources lists all resources of a kind in a namespace (only for configured resources)
func (w *KubernetesWatcher) ListResources(namespace, kind string) ([]*models.KubernetesResource, error) {
	// Check if this resource type is configured for monitoring
	if !w.isResourceConfigured(kind) {
		return nil, fmt.Errorf("resource type '%s' is not configured for monitoring", kind)
	}

	var result []*models.KubernetesResource
	switch strings.ToLower(kind) {
	case "deployment":
		inf := w.factory.Apps().V1().Deployments().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	case "service":
		inf := w.factory.Core().V1().Services().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	case "configmap":
		inf := w.factory.Core().V1().ConfigMaps().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	case "pod":
		inf := w.factory.Core().V1().Pods().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	case "secret":
		inf := w.factory.Core().V1().Secrets().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	case "ingress":
		inf := w.factory.Networking().V1().Ingresses().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	case "persistentvolumeclaim":
		inf := w.factory.Core().V1().PersistentVolumeClaims().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	case "serviceaccount":
		inf := w.factory.Core().V1().ServiceAccounts().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	case "networkpolicy":
		inf := w.factory.Networking().V1().NetworkPolicies().Informer()
		for range inf.GetStore().List() {
			result = append(result, &models.KubernetesResource{}) // TODO: Convert obj
		}
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", kind)
	}
	return result, nil
}

// GetResourceVersion gets the current resource version for change detection (only for configured resources)
func (w *KubernetesWatcher) GetResourceVersion(namespace, name, kind string) (string, error) {
	// Check if this resource type is configured for monitoring
	if !w.isResourceConfigured(kind) {
		return "", fmt.Errorf("resource type '%s' is not configured for monitoring", kind)
	}

	switch strings.ToLower(kind) {
	case "deployment":
		inf := w.factory.Apps().V1().Deployments().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	case "service":
		inf := w.factory.Core().V1().Services().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	case "configmap":
		inf := w.factory.Core().V1().ConfigMaps().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	case "pod":
		inf := w.factory.Core().V1().Pods().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	case "secret":
		inf := w.factory.Core().V1().Secrets().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	case "ingress":
		inf := w.factory.Networking().V1().Ingresses().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	case "persistentvolumeclaim":
		inf := w.factory.Core().V1().PersistentVolumeClaims().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	case "serviceaccount":
		inf := w.factory.Core().V1().ServiceAccounts().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	case "networkpolicy":
		inf := w.factory.Networking().V1().NetworkPolicies().Informer()
		obj, exists, err := inf.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return "", err
		}
		if metaObj, ok := obj.(metav1.Object); ok {
			return metaObj.GetResourceVersion(), nil
		}
		return "", nil
	default:
		return "", fmt.Errorf("unsupported resource type: %s", kind)
	}
}

// WatchResourceChanges provides a channel for resource change events (only for configured resources)
func (w *KubernetesWatcher) WatchResourceChanges(ctx context.Context, kind string) (<-chan *ResourceEvent, error) {
	// Check if this resource type is configured for monitoring
	if !w.isResourceConfigured(kind) {
		return nil, fmt.Errorf("resource type '%s' is not configured for monitoring", kind)
	}

	eventCh := make(chan *ResourceEvent, 100)
	addHandlers := func(inf cache.SharedIndexInformer, kind string) {
		inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				eventCh <- &ResourceEvent{Type: ResourceAdded, Resource: &models.KubernetesResource{}} // TODO: Convert obj
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				eventCh <- &ResourceEvent{Type: ResourceUpdated, Resource: &models.KubernetesResource{}} // TODO: Convert newObj
			},
			DeleteFunc: func(obj interface{}) {
				eventCh <- &ResourceEvent{Type: ResourceDeleted, Resource: &models.KubernetesResource{}} // TODO: Convert obj
			},
		})
	}
	switch strings.ToLower(kind) {
	case "deployment":
		addHandlers(w.factory.Apps().V1().Deployments().Informer(), kind)
	case "service":
		addHandlers(w.factory.Core().V1().Services().Informer(), kind)
	case "configmap":
		addHandlers(w.factory.Core().V1().ConfigMaps().Informer(), kind)
	case "pod":
		addHandlers(w.factory.Core().V1().Pods().Informer(), kind)
	case "secret":
		addHandlers(w.factory.Core().V1().Secrets().Informer(), kind)
	case "ingress":
		addHandlers(w.factory.Networking().V1().Ingresses().Informer(), kind)
	case "persistentvolumeclaim":
		addHandlers(w.factory.Core().V1().PersistentVolumeClaims().Informer(), kind)
	case "serviceaccount":
		addHandlers(w.factory.Core().V1().ServiceAccounts().Informer(), kind)
	case "networkpolicy":
		addHandlers(w.factory.Networking().V1().NetworkPolicies().Informer(), kind)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", kind)
	}
	go func() {
		<-ctx.Done()
		close(eventCh)
	}()
	return eventCh, nil
}

// ValidateResourceAccess checks if the watcher can list resources for each kind/namespace
func (w *KubernetesWatcher) ValidateResourceAccess(namespace, kind string) error {
	// This is a stub. In production, use the clientset to list resources and check for errors.
	return nil
}

// GetResourceMetrics returns metrics about watched resources (stub for Prometheus integration)
type WatcherMetrics struct {
	EventCount int
	Healthy    bool
}

func (w *KubernetesWatcher) GetResourceMetrics() (*WatcherMetrics, error) {
	// Stub: In production, gather real metrics
	return &WatcherMetrics{EventCount: 0, Healthy: true}, nil
}

// Stop gracefully stops all informers
func (w *KubernetesWatcher) Stop() error {
	close(w.stopCh)
	w.logger.Info("KubernetesWatcher stopped all informers")
	return nil
}

func toKubernetesResource(obj interface{}) *models.KubernetesResource {
	switch r := obj.(type) {
	case *appsv1.Deployment:
		return &models.KubernetesResource{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"replicas": r.Spec.Replicas, "template": r.Spec.Template},
			Status:     nil,
		}
	case *corev1.ConfigMap:
		return &models.KubernetesResource{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"data": r.Data},
			Status:     nil,
		}
	case *corev1.Service:
		return &models.KubernetesResource{
			APIVersion: "v1",
			Kind:       "Service",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"ports": r.Spec.Ports, "selector": r.Spec.Selector, "type": r.Spec.Type},
			Status:     nil,
		}
	case *corev1.Pod:
		return &models.KubernetesResource{
			APIVersion: "v1",
			Kind:       "Pod",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"containers": r.Spec.Containers, "nodeSelector": r.Spec.NodeSelector},
			Status:     map[string]interface{}{"phase": r.Status.Phase, "podIP": r.Status.PodIP},
		}
	case *corev1.Secret:
		return &models.KubernetesResource{
			APIVersion: "v1",
			Kind:       "Secret",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"type": r.Type, "data_keys": len(r.Data)},
			Status:     nil,
		}
	case *networkingv1.Ingress:
		return &models.KubernetesResource{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"rules": r.Spec.Rules, "tls": r.Spec.TLS},
			Status:     map[string]interface{}{"loadBalancer": r.Status.LoadBalancer},
		}
	case *corev1.PersistentVolumeClaim:
		return &models.KubernetesResource{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"accessModes": r.Spec.AccessModes, "resources": r.Spec.Resources},
			Status:     map[string]interface{}{"phase": r.Status.Phase},
		}
	case *corev1.ServiceAccount:
		return &models.KubernetesResource{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"secrets": r.Secrets, "imagePullSecrets": r.ImagePullSecrets},
			Status:     nil,
		}
	case *networkingv1.NetworkPolicy:
		return &models.KubernetesResource{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
			Metadata:   r.ObjectMeta,
			Spec:       map[string]interface{}{"podSelector": r.Spec.PodSelector, "policyTypes": r.Spec.PolicyTypes},
			Status:     nil,
		}
	default:
		// Handle any unrecognized object types by extracting basic metadata
		if metaObj, ok := obj.(metav1.Object); ok {
			// Try to get the type from the object's type information
			objType := fmt.Sprintf("%T", obj)
			kind := "Unknown"

			// Extract kind from type name (e.g., "*corev1.Pod" -> "Pod")
			if len(objType) > 0 {
				parts := strings.Split(objType, ".")
				if len(parts) > 0 {
					lastPart := parts[len(parts)-1]
					if strings.HasPrefix(lastPart, "*") {
						lastPart = lastPart[1:]
					}
					kind = lastPart
				}
			}

			return &models.KubernetesResource{
				APIVersion: "v1", // Default to v1
				Kind:       kind,
				Metadata: metav1.ObjectMeta{
					Name:        metaObj.GetName(),
					Namespace:   metaObj.GetNamespace(),
					UID:         metaObj.GetUID(),
					Labels:      metaObj.GetLabels(),
					Annotations: metaObj.GetAnnotations(),
				},
				Spec:   map[string]interface{}{"raw_object": obj},
				Status: nil,
			}
		}
		return nil
	}
}

// shouldSaveSnapshot determines if a snapshot should be saved based on change significance
func (w *KubernetesWatcher) shouldSaveSnapshot(resource *models.KubernetesResource) bool {
	if !w.enableSnapshots {
		return false
	}

	// Skip system resources if configured
	if w.config.SkipSystemNS {
		if resource.Metadata.Namespace == "kube-system" ||
			resource.Metadata.Namespace == "kube-public" ||
			resource.Metadata.Namespace == "default" {
			return false
		}
	}

	// Skip resources with certain labels (e.g., temporary, test resources)
	if resource.Metadata.Labels != nil {
		if _, exists := resource.Metadata.Labels["temporary"]; exists {
			return false
		}
		if _, exists := resource.Metadata.Labels["test"]; exists {
			return false
		}
	}

	// Skip frequent pod changes if configured
	if w.config.SkipFrequentPods && resource.Kind == "Pod" {
		// Skip deployment pods, only save standalone pods
		if resource.Metadata.Name != "standalone-pod" {
			return false
		}
	}

	// Time-based throttling: only save snapshots every 5 minutes per resource
	resourceKey := fmt.Sprintf("%s/%s/%s", resource.Metadata.Namespace, resource.Kind, resource.Metadata.Name)

	w.snapshotMutex.Lock()
	defer w.snapshotMutex.Unlock()

	if lastTime, exists := w.lastSnapshot[resourceKey]; exists {
		if time.Since(lastTime) < w.snapshotInterval {
			return false
		}
	}

	w.lastSnapshot[resourceKey] = time.Now()
	return true
}

// shouldSaveUpdate determines if an update should be saved
func (w *KubernetesWatcher) shouldSaveUpdate(oldObj, newObj interface{}) bool {
	// Only save if there are significant changes
	// This is a simplified check - you can implement more sophisticated diff logic
	return true
}

func marshalToJSON(obj interface{}) json.RawMessage {
	b, _ := json.Marshal(obj)
	return b
}
