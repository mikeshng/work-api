/*
Copyright 2022 The Kubernetes Authors.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workv1alpha1 "sigs.k8s.io/work-api/pkg/apis/v1alpha1"
	"sigs.k8s.io/work-api/pkg/statusfeedback"
)

const statusFeedbackConditionType = "StatusFeedbackSynced"

// StatusFeedbackReconciler reconciles a Work object for finalization
type StatusFeedbackReconciler struct {
	client             client.Client
	spokeDynamicClient dynamic.Interface
	restMapper         meta.RESTMapper
	log                logr.Logger
	statusSyncInterval time.Duration
	statusReader       *statusfeedback.StatusReader
}

// Reconcile implement the control loop logic for finalizing Work object.
func (r *StatusFeedbackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log.Info("Reconciling " + req.Name) // TODO fix this logging

	originalWork := &workv1alpha1.Work{}
	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, originalWork)
	switch {
	case errors.IsNotFound(err):
		return ctrl.Result{}, nil
	case err != nil:
		return ctrl.Result{}, err
	}

	work := originalWork.DeepCopy()

	for _, manifest := range work.Spec.Workload.Manifests {
		if manifest.StatusFeedbackRules == nil {
			continue
		}

		resource, err := r.getResourceObject(manifest)
		if err != nil {
			r.log.Error(err, "failed to get resource object")

			return ctrl.Result{RequeueAfter: r.statusSyncInterval}, nil
		}

		// Read status of the resource according to feedback rules.
		values, statusFeedbackCondition := r.getFeedbackValues(resource, manifest.StatusFeedbackRules)

		for index, manifestCondition := range work.Status.ManifestConditions {
			identifier := manifestCondition.Identifier
			gvk := resource.GroupVersionKind()

			if identifier.Namespace == resource.GetNamespace() && identifier.Name == resource.GetName() &&
				identifier.Group == gvk.Group && identifier.Version == gvk.Version && identifier.Kind == gvk.Kind {
				meta.SetStatusCondition(&work.Status.ManifestConditions[index].Conditions, statusFeedbackCondition)
				work.Status.ManifestConditions[index].StatusFeedbacks.Values = values

				break
			}
		}
	}

	// don't do anything if the status of work did not change
	if equality.Semantic.DeepEqual(originalWork.Status.Conditions, work.Status.Conditions) &&
		equality.Semantic.DeepEqual(originalWork.Status.ManifestConditions, work.Status.ManifestConditions) {
		return ctrl.Result{RequeueAfter: r.statusSyncInterval}, nil
	}

	// update status of work. if this conflicts, try again later
	err = r.client.Status().Update(ctx, work, &client.UpdateOptions{})

	return ctrl.Result{RequeueAfter: r.statusSyncInterval}, err
}

func (c *StatusFeedbackReconciler) getFeedbackValues(obj *unstructured.Unstructured,
	statusFeedbackRules []workv1alpha1.StatusFeedbackRule) ([]workv1alpha1.FeedbackValue, metav1.Condition) {
	errs := []error{}
	values := []workv1alpha1.FeedbackValue{}

	for _, rule := range statusFeedbackRules {
		valuesByRule, err := c.statusReader.GetValuesByRule(obj, rule)
		if err != nil {
			errs = append(errs, err)
		}
		if len(valuesByRule) > 0 {
			values = append(values, valuesByRule...)
		}
	}

	err := utilerrors.NewAggregate(errs)

	if err != nil {
		return values, metav1.Condition{
			Type:    statusFeedbackConditionType,
			Reason:  "StatusFeedbackSyncFailed",
			Status:  metav1.ConditionFalse,
			Message: fmt.Sprintf("Sync status feedback failed with error %v", err),
		}
	}

	if len(values) == 0 {
		return values, metav1.Condition{
			Type:   statusFeedbackConditionType,
			Reason: "NoStatusFeedbackSynced",
			Status: metav1.ConditionTrue,
		}
	}

	return values, metav1.Condition{
		Type:   statusFeedbackConditionType,
		Reason: "StatusFeedbackSynced",
		Status: metav1.ConditionTrue,
	}
}

// getResourceObject returns a resource object given the manifest
func (c *StatusFeedbackReconciler) getResourceObject(manifest workv1alpha1.Manifest) (
	*unstructured.Unstructured, error) {
	gvr, unstructuredObj, err := decodeUnstructured(manifest, c.restMapper)
	if err != nil {
		return nil, err
	}

	return c.spokeDynamicClient.Resource(gvr).Namespace(unstructuredObj.GetNamespace()).
		Get(context.TODO(), unstructuredObj.GetName(), metav1.GetOptions{})
}

// SetupWithManager wires up the controller.
func (r *StatusFeedbackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&workv1alpha1.Work{}).Complete(r)
}
