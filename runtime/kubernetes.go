package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/yurifrl/tennon/domain"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Kubernetes runs connector workloads as Pods in a Kubernetes cluster and
// returns their reachable address for proxying. Credentials are passed as env
// vars from the rendered headers.
type Kubernetes struct {
	client    client.Client
	namespace string
	mu        sync.Mutex
	pods      map[string]podInfo // connector name -> pod info
}

type podInfo struct {
	name    string
	address string
}

// NewKubernetes creates a Kubernetes runtime using the provided client.
// namespace is where connector Pods will be created (defaults to "tennon").
func NewKubernetes(c client.Client, namespace string) *Kubernetes {
	if namespace == "" {
		namespace = "tennon"
	}
	return &Kubernetes{
		client:    c,
		namespace: namespace,
		pods:      make(map[string]podInfo),
	}
}

func (*Kubernetes) Supports(kind string) bool { return kind == "container" }

func (k *Kubernetes) Ensure(ctx context.Context, c *domain.Connector, headers map[string]string) (*domain.Endpoint, error) {
	if c.Spec.Runtime.Container == nil || c.Spec.Runtime.Container.Image == "" {
		return nil, fmt.Errorf("runtime/kubernetes: connector %q has no container.image", c.Meta.Name)
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	podName := "tennon-" + c.Meta.Name

	// Check if pod already exists and is running
	if info, ok := k.pods[c.Meta.Name]; ok {
		pod := &corev1.Pod{}
		err := k.client.Get(ctx, client.ObjectKey{Namespace: k.namespace, Name: info.name}, pod)
		if err == nil && pod.Status.Phase == corev1.PodRunning {
			return k.endpoint(info.address, c), nil
		}
		// Pod doesn't exist or isn't running, remove from cache
		delete(k.pods, c.Meta.Name)
	}

	// Create the pod
	port := int32(8080)
	if p := c.Meta.Labels["tennon.io/port"]; p != "" {
		// Parse port if custom one is specified
		var portInt int
		if _, err := fmt.Sscanf(p, "%d", &portInt); err == nil {
			port = int32(portInt)
		}
	}

	// Build env vars from headers and spec.Env
	envVars := []corev1.EnvVar{}
	for k, v := range headers {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}
	for k, v := range c.Spec.Env {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":                       "tennon",
				"tennon.io/connector":       c.Meta.Name,
				"tennon.io/connector-type":  "container",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "mcp",
					Image: c.Spec.Runtime.Container.Image,
					Ports: []corev1.ContainerPort{
						{
							Name:          "mcp",
							ContainerPort: port,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Env: envVars,
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	// Create the pod
	if err := k.client.Create(ctx, pod); err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("runtime/kubernetes: create pod %q: %w", podName, err)
	}

	// Wait for pod to be ready
	// In a real implementation, this would be done by a controller watching pod status
	// For now, we'll construct the service endpoint
	serviceName := podName
	address := fmt.Sprintf("%s.%s.svc.cluster.local:%d", serviceName, k.namespace, port)

	// Create a Service to expose the pod
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":                "tennon",
				"tennon.io/connector": c.Meta.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"tennon.io/connector": c.Meta.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "mcp",
					Port:       port,
					TargetPort: intstr.FromInt32(port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if err := k.client.Create(ctx, svc); err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("runtime/kubernetes: create service %q: %w", serviceName, err)
	}

	k.pods[c.Meta.Name] = podInfo{name: podName, address: address}
	return k.endpoint(address, c), nil
}

func (k *Kubernetes) Teardown(ctx context.Context, c *domain.Connector) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	podName := "tennon-" + c.Meta.Name
	serviceName := podName

	// Delete the service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: k.namespace,
		},
	}
	if err := k.client.Delete(ctx, svc); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("runtime/kubernetes: delete service %q: %w", serviceName, err)
	}

	// Delete the pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: k.namespace,
		},
	}
	if err := k.client.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("runtime/kubernetes: delete pod %q: %w", podName, err)
	}

	delete(k.pods, c.Meta.Name)
	return nil
}

func (k *Kubernetes) endpoint(address string, c *domain.Connector) *domain.Endpoint {
	transport := c.Spec.Transport
	if transport == "" {
		transport = "streamable-http"
	}
	return &domain.Endpoint{
		URL:       fmt.Sprintf("http://%s/mcp", address),
		Transport: transport,
	}
}
