package deploy

import (
	"context"
	"fmt"

	"github.com/dcm-project/dcm-placement-api/internal/catalog"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type ContainerService struct {
	client kubernetes.Interface
}

func NewContainerService(client kubernetes.Interface) *ContainerService {
	return &ContainerService{
		client: client,
	}
}

func (c *ContainerService) HandleContainerDeployment(ctx context.Context, app *catalog.ContainerApp, name, namespace string, id string) error {
	logger := zap.S().Named("container_service:deployment_handler")
	logger.Info("Starting deployment for Container")

	// Create Namespace for the Virtual Machine
	if err := c.checkOrCreateNamespace(ctx, namespace); err != nil {
		return err
	}

	// Deploy container
	if err := c.deployContainer(ctx, name, namespace, app, id); err != nil {
		return err
	}

	// Deploy service
	err := c.deployContainerService(ctx, name, namespace, app, id)
	if err != nil {
		return err
	}
	logger.Info("Successfully deployed container")
	return nil
}

func (c *ContainerService) checkOrCreateNamespace(ctx context.Context, namespace string) error {
	logger := zap.S().Named("container_service:check_Namespace")
	// Check Namespace exists
	_, err := c.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// Create Namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = c.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			logger.Error("Error occurred when creating namespace", err)
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
	}
	logger.Info("Successfully created namespace.", " Namespace ", namespace)
	return nil
}

func (c *ContainerService) deployContainer(ctx context.Context, name, namespace string, app *catalog.ContainerApp, id string) error {
	logger := zap.S().Named("container_service:deploy_container")

	// Create deployment object
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", name),
			Labels: map[string]string{
				"app-id": id,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &app.Replica,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: app.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: int32(app.Port)},
							},
						},
					},
				},
			},
		},
	}

	// Apply deployment object to cluster
	_, err := c.client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		logger.Error("failed to deploy container", "Name", name)
		return err
	}
	logger.Info("Successfully deployed container")
	return nil
}

func (c *ContainerService) deployContainerService(ctx context.Context, name, namespace string, app *catalog.ContainerApp, id string) error {
	logger := zap.S().Named("container_service:deploy_service")

	// Create Service object
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-service-", name),
			Labels: map[string]string{
				"app-id": id,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"service": name},
			Ports: []corev1.ServicePort{
				{
					Port:       int32(app.Port),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(int32(app.Port)),
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	_, err := c.client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		logger.Error("failed to create service for application")
		return err
	}
	logger.Info("Successfully created service for container: ", "Name", name)
	return nil
}
