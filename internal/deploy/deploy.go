package deploy

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"github.com/dcm-project/dcm-placement-api/internal/catalog"
)

type DeployService struct {
	client kubecli.KubevirtClient
}

func NewDeployService(client kubecli.KubevirtClient) *DeployService {
	return &DeployService{
		client: client,
	}
}

func (s *DeployService) DeleteVM(ctx context.Context, id string, namespace string) error {
	logger := zap.S().Named("delete_vm")
	logger.Info("Starting deletion for Virtual Machine")
	err := s.client.VirtualMachine(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app-id=%s", id),
	})
	if err != nil {
		return fmt.Errorf("failed to delete VirtualMachine: %w", err)
	}
	logger.Info("Virtual Machine deleted successfully")
	return nil
}

func (s *DeployService) DeployVM(ctx context.Context, name string, vm *catalog.CatalogVm, namespace string, id string) error {
	logger := zap.S().Named("deploy_vm")
	logger.Info("Starting deployment for Virtual Machine")
	// Create Namespace for the Virtual Machine
	if err := s.getNamespace(ctx, namespace); err != nil {
		return err
	}

	// Create the VirtualMachine object
	memory := resource.MustParse(fmt.Sprintf("%dGi", vm.Ram))
	virtualMachine := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", name),
			Namespace:    namespace,
			Labels: map[string]string{
				"app-id": id,
			},
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			RunStrategy: &[]kubevirtv1.VirtualMachineRunStrategy{kubevirtv1.RunStrategyRerunOnFailure}[0],
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					AccessCredentials: []kubevirtv1.AccessCredential{
						{
							SSHPublicKey: &kubevirtv1.SSHPublicKeyAccessCredential{
								Source: kubevirtv1.SSHPublicKeyAccessCredentialSource{
									Secret: &kubevirtv1.AccessCredentialSecretSource{
										SecretName: "myssh",
									},
								},
								PropagationMethod: kubevirtv1.SSHPublicKeyAccessCredentialPropagationMethod{
									NoCloud: &kubevirtv1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
								},
							},
						},
					},
					Architecture: "amd64",
					Domain: kubevirtv1.DomainSpec{
						CPU: &kubevirtv1.CPU{
							Cores: uint32(vm.Cpu),
						},
						Memory: &kubevirtv1.Memory{
							Guest: &memory,
						},
						Devices: kubevirtv1.Devices{
							Disks: []kubevirtv1.Disk{
								{
									Name:      fmt.Sprintf("%s-disk", name),
									BootOrder: &[]uint{1}[0],
									DiskDevice: kubevirtv1.DiskDevice{
										Disk: &kubevirtv1.DiskTarget{
											Bus: kubevirtv1.DiskBusVirtio,
										},
									},
								},
								{
									Name:      "cloudinitdisk",
									BootOrder: &[]uint{2}[0],
									DiskDevice: kubevirtv1.DiskDevice{
										Disk: &kubevirtv1.DiskTarget{
											Bus: kubevirtv1.DiskBusVirtio,
										},
									},
								},
							},
							Interfaces: []kubevirtv1.Interface{
								{
									Name: "myvmnic",
									InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
										Bridge: &kubevirtv1.InterfaceBridge{},
									},
								},
							},
							Rng: &kubevirtv1.Rng{},
						},
						Features: &kubevirtv1.Features{
							ACPI: kubevirtv1.FeatureState{},
							SMM: &kubevirtv1.FeatureState{
								Enabled: &[]bool{true}[0],
							},
						},
						Machine: &kubevirtv1.Machine{
							Type: "pc-q35-rhel9.6.0",
						},
					},
					Networks: []kubevirtv1.Network{
						{
							Name: "myvmnic",
							NetworkSource: kubevirtv1.NetworkSource{
								Pod: &kubevirtv1.PodNetwork{},
							},
						},
					},
					TerminationGracePeriodSeconds: &[]int64{180}[0],
					Volumes: []kubevirtv1.Volume{
						{
							Name: fmt.Sprintf("%s-disk", name),
							VolumeSource: kubevirtv1.VolumeSource{
								ContainerDisk: &kubevirtv1.ContainerDiskSource{
									Image: s.getOSImage(vm.Os),
								},
							},
						},
						{
							Name: "cloudinitdisk",
							VolumeSource: kubevirtv1.VolumeSource{
								CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
									UserData: s.generateCloudInitUserData(name, vm),
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the VirtualMachine in the cluster
	_, err := s.client.VirtualMachine(namespace).Create(ctx, virtualMachine, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create VirtualMachine: %w", err)
	}

	return nil
}

// getNamespace checks and creates namespace
func (s *DeployService) getNamespace(ctx context.Context, namespace string) error {
	logger := zap.S().Named("get_namespace")
	// Check Namespace exists
	_, err := s.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// Create Namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = s.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			logger.Error("Error occurred when creating namespace", err)
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
	}
	logger.Info("Successfully created namespace.", "Namespace", namespace)
	return nil
}

// getOSImage returns the container image for the specified OS
func (s *DeployService) getOSImage(os string) string {
	images := map[string]string{
		"fedora": "quay.io/containerdisks/fedora:latest",
		"ubuntu": "quay.io/containerdisks/ubuntu:latest",
		"centos": "quay.io/containerdisks/centos:latest",
		"rhel":   "quay.io/containerdisks/rhel:latest",
	}

	if image, exists := images[os]; exists {
		return image
	}
	// Default to fedora if OS not found
	return "quay.io/containerdisks/fedora:latest"
}

// generateCloudInitUserData generates cloud-init user data for the VM
func (s *DeployService) generateCloudInitUserData(appName string, vm *catalog.CatalogVm) string {
	return fmt.Sprintf(`#cloud-config
user: %s
password: auto-generated-pass
chpasswd: { expire: False }
hostname: %s
`, vm.Os, appName)
}
