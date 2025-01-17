/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tektoninstallerset

import (
	"errors"
	"strings"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	namespacePred                      = mf.ByKind("Namespace")
	configMapPred                      = mf.ByKind("ConfigMap")
	secretPred                         = mf.ByKind("Secret")
	deploymentPred                     = mf.ByKind("Deployment")
	servicePred                        = mf.ByKind("Service")
	serviceAccountPred                 = mf.ByKind("ServiceAccount")
	rolePred                           = mf.ByKind("Role")
	roleBindingPred                    = mf.ByKind("RoleBinding")
	clusterRolePred                    = mf.ByKind("ClusterRole")
	clusterRoleBindingPred             = mf.ByKind("ClusterRoleBinding")
	podSecurityPolicyPred              = mf.ByKind("PodSecurityPolicy")
	validatingWebhookConfigurationPred = mf.ByKind("ValidatingWebhookConfiguration")
	mutatingWebhookConfigurationPred   = mf.ByKind("MutatingWebhookConfiguration")
	horizontalPodAutoscalerPred        = mf.ByKind("HorizontalPodAutoscaler")
)

type installer struct {
	Manifest mf.Manifest
}

func (i *installer) EnsureCRDs() error {
	if err := i.Manifest.Filter(mf.Any(mf.CRDs)).Apply(); err != nil {
		return err
	}
	return nil
}

func (i *installer) EnsureClusterScopedResources() error {
	if err := i.Manifest.Filter(
		mf.Any(
			namespacePred,
			clusterRolePred,
			podSecurityPolicyPred,
			validatingWebhookConfigurationPred,
			mutatingWebhookConfigurationPred,
		)).Apply(); err != nil {
		return err
	}
	return nil
}

func (i *installer) EnsureNamespaceScopedResources() error {
	if err := i.Manifest.Filter(
		mf.Any(
			serviceAccountPred,
			clusterRoleBindingPred,
			rolePred,
			roleBindingPred,
			configMapPred,
			secretPred,
			horizontalPodAutoscalerPred,
		)).Apply(); err != nil {
		return err
	}
	return nil
}

func (i *installer) EnsureDeploymentResources() error {
	if err := i.Manifest.Filter(
		mf.Any(
			deploymentPred,
			servicePred,
		)).Apply(); err != nil {
		return err
	}
	return nil
}

func (i *installer) IsWebhookReady() error {
	return i.IsDeploymentReady("webhook")
}

func (i *installer) IsControllerReady() error {
	return i.IsDeploymentReady("controller")
}

func (i *installer) IsDeploymentReady(name string) error {

	for _, u := range i.Manifest.Filter(deploymentPred).Resources() {

		if !strings.Contains(u.GetName(), name) {
			continue
		}

		resource, err := i.Manifest.Client.Get(&u)
		if err != nil {
			return err
		}

		deployment := &appsv1.Deployment{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, deployment)
		if err != nil {
			return err
		}

		if !isDeploymentAvailable(deployment) {
			return errors.New("deployment not available")
		}
	}

	return nil
}

func isDeploymentAvailable(d *appsv1.Deployment) bool {
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
