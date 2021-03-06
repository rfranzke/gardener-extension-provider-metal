package infrastructure

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	"github.com/pkg/errors"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenerkubernetes "github.com/gardener/gardener/pkg/client/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type actuator struct {
	logger logr.Logger

	clientset         kubernetes.Interface
	gardenerClientset gardenerkubernetes.Interface
	restConfig        *rest.Config

	client  client.Client
	scheme  *runtime.Scheme
	decoder runtime.Decoder
}

// NewActuator creates a new Actuator that updates the status of the handled Infrastructure resources.
func NewActuator() infrastructure.Actuator {
	return &actuator{
		logger: log.Log.WithName("infrastructure-actuator"),
	}
}

func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	a.decoder = serializer.NewCodecFactory(a.scheme).UniversalDecoder()
	return nil
}

func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

func (a *actuator) InjectConfig(config *rest.Config) error {
	var err error
	a.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "could not create Kubernetes client")
	}

	a.gardenerClientset, err = gardenerkubernetes.NewWithConfig(gardenerkubernetes.WithRESTConfig(config))
	if err != nil {
		return errors.Wrap(err, "could not create Gardener client")
	}

	a.restConfig = config
	return nil
}

func decodeInfrastructure(infrastructure *extensionsv1alpha1.Infrastructure, decoder runtime.Decoder) (*metalapi.InfrastructureConfig, *metalapi.InfrastructureStatus, error) {
	infrastructureConfig, err := helper.InfrastructureConfigFromInfrastructure(infrastructure)
	if err != nil {
		return nil, nil, err
	}

	infrastructureStatus := &metalapi.InfrastructureStatus{}
	if infrastructure.Status.ProviderStatus != nil {
		if _, _, err := decoder.Decode(infrastructure.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
			return nil, nil, fmt.Errorf("could not decode infrastructure status: %+v", err)
		}
	}

	return infrastructureConfig, infrastructureStatus, nil
}

func updateProviderStatus(ctx context.Context, c client.Client, infrastructure *extensionsv1alpha1.Infrastructure, providerStatus *metalapi.InfrastructureStatus, nodeCIDR *string) error {
	return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, c, infrastructure, func() error {
		infrastructure.Status.ProviderStatus = &runtime.RawExtension{Object: &metalv1alpha1.InfrastructureStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: metalv1alpha1.SchemeGroupVersion.String(),
				Kind:       "InfrastructureStatus",
			},
			Firewall: metalv1alpha1.FirewallStatus{
				Succeeded: providerStatus.Firewall.Succeeded,
				MachineID: providerStatus.Firewall.MachineID,
			},
		}}
		infrastructure.Status.NodesCIDR = nodeCIDR
		return nil
	})
}
