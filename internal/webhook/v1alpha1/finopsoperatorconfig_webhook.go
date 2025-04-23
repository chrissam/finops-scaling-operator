/*
   Copyright 2025.

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

   package v1alpha1

   import (
       "context"
       "fmt"
	   "errors"

       "k8s.io/apimachinery/pkg/runtime"
       ctrl "sigs.k8s.io/controller-runtime"
       logf "sigs.k8s.io/controller-runtime/pkg/log"
       "sigs.k8s.io/controller-runtime/pkg/webhook"
       "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
       "sigs.k8s.io/controller-runtime/pkg/client"

       finopsv1alpha1 "github.com/chrissam/k8s-scaling-operator/api/v1alpha1"
   )

   // nolint:unused
   // log is for logging in this package.
   var finopsoperatorconfiglog = logf.Log.WithName("finopsoperatorconfig-resource")
   var webhookClient client.Client //  Add client for listing resources

   // SetupFinOpsOperatorConfigWebhookWithManager registers the webhook for FinOpsOperatorConfig in the manager.
   func SetupFinOpsOperatorConfigWebhookWithManager(mgr ctrl.Manager) error {
       webhookClient = mgr.GetClient() //  Initialize the client
       return ctrl.NewWebhookManagedBy(mgr).
           For(&finopsv1alpha1.FinOpsOperatorConfig{}).
           WithValidator(&FinOpsOperatorConfigCustomValidator{}).
           Complete()
   }

   // TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

   // TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
   // NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
   // Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
   // +kubebuilder:webhook:path=/validate-finops-devopsideas-com-v1alpha1-finopsoperatorconfig,mutating=false,failurePolicy=Fail,sideEffects=None,groups=finops.devopsideas.com,resources=finopsoperatorconfigs,verbs=create;update,versions=v1alpha1,name=vfinopsoperatorconfig-v1alpha1.kb.io,admissionReviewVersions=v1

   // FinOpsOperatorConfigCustomValidator struct is responsible for validating the FinOpsOperatorConfig resource
   // when it is created, updated, or deleted.
   //
   // NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
   // as this struct is used only for temporary operations and does not need to be deeply copied.
   type FinOpsOperatorConfigCustomValidator struct {
       // TODO(user): Add more fields as needed for validation
   }

   var _ webhook.CustomValidator = &FinOpsOperatorConfigCustomValidator{}

   // ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type FinOpsOperatorConfig.
   func (v *FinOpsOperatorConfigCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
       finopsoperatorconfig, ok := obj.(*finopsv1alpha1.FinOpsOperatorConfig)
       if !ok {
           return nil, fmt.Errorf("expected a FinOpsOperatorConfig object but got %T", obj)
       }
       finopsoperatorconfiglog.Info("Validation for FinOpsOperatorConfig upon creation", "name", finopsoperatorconfig.GetName())

       //  Check if any FinOpsOperatorConfig resources already exist
       configList := &finopsv1alpha1.FinOpsOperatorConfigList{}
       if err := webhookClient.List(ctx, configList); err != nil {
           finopsoperatorconfiglog.Error(err, "Failed to list existing FinOpsOperatorConfig resources")
           return nil, errors.New("internal error: failed to list existing FinOpsOperatorConfig resources")
       }

       if len(configList.Items) > 0 {
           return nil, errors.New("only one FinOpsOperatorConfig resource can be created in the cluster")
       }

       return nil, nil
   }

   // ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type FinOpsOperatorConfig.
   func (v *FinOpsOperatorConfigCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
       finopsoperatorconfig, ok := newObj.(*finopsv1alpha1.FinOpsOperatorConfig)
       if !ok {
           return nil, fmt.Errorf("expected a FinOpsOperatorConfig object for the newObj but got %T", newObj)
       }
       finopsoperatorconfiglog.Info("Validation for FinOpsOperatorConfig upon update", "name", finopsoperatorconfig.GetName())

       //  Check if more than one FinOpsOperatorConfig resource exists (should not happen on update, but a safety check)
       configList := &finopsv1alpha1.FinOpsOperatorConfigList{}
       if err := webhookClient.List(ctx, configList); err != nil {
           finopsoperatorconfiglog.Error(err, "Failed to list existing FinOpsOperatorConfig resources")
           return nil, errors.New("internal error: failed to list existing FinOpsOperatorConfig resources")
       }

       if len(configList.Items) > 1 {
           return nil, errors.New("only one FinOpsOperatorConfig resource is allowed in the cluster")
       }

       return nil, nil
   }

   // ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type FinOpsOperatorConfig.
   func (v *FinOpsOperatorConfigCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
       finopsoperatorconfig, ok := obj.(*finopsv1alpha1.FinOpsOperatorConfig)
       if !ok {
           return nil, fmt.Errorf("expected a FinOpsOperatorConfig object but got %T", obj)
       }
       finopsoperatorconfiglog.Info("Validation for FinOpsOperatorConfig upon deletion", "name", finopsoperatorconfig.GetName())

       //  Add any specific deletion validation logic here (if needed)
       //  For example, you might want to prevent deletion under certain conditions

       return nil, nil
   }