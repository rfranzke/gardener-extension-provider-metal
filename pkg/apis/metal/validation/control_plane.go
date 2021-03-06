package validation

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateControlPlaneConfig validates a ControlPlaneConfig object.
func ValidateControlPlaneConfig(controlPlaneConfig *apismetal.ControlPlaneConfig, cloudProfile *gardencorev1beta1.CloudProfile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	iam := controlPlaneConfig.IAMConfig
	iamPath := fldPath.Child("iamconfig")
	if iam == nil {
		allErrs = append(allErrs, field.Required(iamPath, "iam config must be specified"))
		return allErrs
	}

	issuer := iam.IssuerConfig
	issuerPath := iamPath.Child("issuerConfig")
	if issuer == nil {
		allErrs = append(allErrs, field.Required(issuerPath, "issuer config must be specified"))
	} else {
		if issuer.Url == "" {
			allErrs = append(allErrs, field.Required(issuerPath.Child("url"), "url must be specified"))
		}
		if issuer.ClientId == "" {
			allErrs = append(allErrs, field.Required(issuerPath.Child("clientId"), "clientId must be specified"))
		}
	}

	groupConfig := iam.GroupConfig
	groupConfigPath := iamPath.Child("groupConfig")
	if groupConfig != nil {
		if groupConfig.NamespaceMaxLength <= 0 {
			allErrs = append(allErrs, field.Required(groupConfigPath.Child("namespaceMaxLength"), "namespaceMaxLength must be a positive integer"))
		}
	}

	return allErrs
}
