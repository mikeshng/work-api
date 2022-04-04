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
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workv1alpha1 "sigs.k8s.io/work-api/pkg/apis/v1alpha1"
	"sigs.k8s.io/work-api/pkg/statussync"
)

const (
	statusSyncConditionType        = "StatusSynced"
	resourceAvailableConditionType = "Available"
)

// StatusSyncController is to update the available status conditions of both manifests and work.
// It is also used to get the status value based on status sync rule in manifest config.
type StatusSyncController struct {
	client             client.Client
	spokeDynamicClient dynamic.Interface
	restMapper         meta.RESTMapper
	log                logr.Logger
	statusSyncInterval time.Duration
	statusReader       *statussync.StatusReader
}

// SetupWithManager wires up the controller.
func (c *StatusSyncController) SetupWithManager(mgr ctrl.Manager) {
	go wait.Until(func() {
		c.syncAllWorks(context.TODO())
	}, c.statusSyncInterval, context.TODO().Done())
}

func (c *StatusSyncController) syncAllWorks(ctx context.Context) {
	c.log.Info("Reconciling all Works")

	workList := &workv1alpha1.WorkList{}

	err := c.client.List(ctx, workList, &client.ListOptions{LabelSelector: labels.Everything()})
	if err != nil {
		c.log.Error(err, "unable to list work")
	}

	if len(workList.Items) == 0 {
		c.log.Info("no work found")
	}

	for _, work := range workList.Items {
		err = c.syncWork(ctx, work)
		if err != nil {
			c.log.Error(err, "unable to sync work "+work.Name)
		}
	}
}

func (c *StatusSyncController) syncWork(ctx context.Context, originalWork workv1alpha1.Work) error {
	c.log.Info("sync work: " + originalWork.Name)

	work := originalWork.DeepCopy()

	// handle status condition of manifests
	// TODO revist this controller since this might bring races when user change the manifests in spec.
	for index, manifest := range work.Spec.Workload.Manifests {
		obj, availableStatusCondition, err := c.buildAvailableStatusCondition(manifest)
		meta.SetStatusCondition(&work.Status.ManifestConditions[index].Conditions, availableStatusCondition)
		if err != nil {
			// skip getting status values if resource is not available.
			continue
		}

		gvk := obj.GroupVersionKind()

		for _, manifestConfig := range work.Spec.WorkloadConfig.ManifestConfigs {
			identifier := manifestConfig.ResourceIdentifier

			// found matching manifest config to manifest
			if identifier.Group == gvk.Group &&
				identifier.Version == gvk.Version &&
				identifier.Kind == gvk.Kind {
				values, statusSyncCondition := c.getSyncValues(obj, manifestConfig.StatusSyncRules)
				meta.SetStatusCondition(&work.Status.ManifestConditions[index].Conditions, statusSyncCondition)
				work.Status.ManifestConditions[index].StatusSync.Values = values

				break
			}
		}
	}

	// aggregate ManifestConditions and update work status condition
	workAvailableStatusCondition := aggregateManifestConditions(work.Generation, work.Status.ManifestConditions)
	meta.SetStatusCondition(&work.Status.Conditions, workAvailableStatusCondition)

	// don't do anything if the status of work did not change
	if equality.Semantic.DeepEqual(originalWork.Status.Conditions, work.Status.Conditions) &&
		equality.Semantic.DeepEqual(originalWork.Status.ManifestConditions, work.Status.ManifestConditions) {
		return nil
	}

	// update status of work. if this conflicts, try again later based on status sync interval
	return c.client.Status().Update(ctx, work, &client.UpdateOptions{})
}

// aggregateManifestConditions aggregates status conditions of manifests and returns a status
// condition for work
func aggregateManifestConditions(generation int64, manifests []workv1alpha1.ManifestCondition) metav1.Condition {
	available, unavailable, unknown := 0, 0, 0
	for _, manifest := range manifests {
		for _, condition := range manifest.Conditions {
			if condition.Type != resourceAvailableConditionType {
				continue
			}

			switch condition.Status {
			case metav1.ConditionTrue:
				available += 1
			case metav1.ConditionFalse:
				unavailable += 1
			case metav1.ConditionUnknown:
				unknown += 1
			}
		}
	}

	switch {
	case unavailable > 0:
		return metav1.Condition{
			Type:               resourceAvailableConditionType,
			Status:             metav1.ConditionFalse,
			Reason:             "ResourcesNotAvailable",
			ObservedGeneration: generation,
			Message:            fmt.Sprintf("%d of %d resources are not available", unavailable, len(manifests)),
		}
	case unknown > 0:
		return metav1.Condition{
			Type:               resourceAvailableConditionType,
			Status:             metav1.ConditionUnknown,
			Reason:             "ResourcesStatusUnknown",
			ObservedGeneration: generation,
			Message:            fmt.Sprintf("%d of %d resources have unknown status", unknown, len(manifests)),
		}
	case available == 0:
		return metav1.Condition{
			Type:               resourceAvailableConditionType,
			Status:             metav1.ConditionUnknown,
			Reason:             "ResourcesStatusUnknown",
			ObservedGeneration: generation,
			Message:            "cannot get any available resource",
		}
	default:
		return metav1.Condition{
			Type:               resourceAvailableConditionType,
			Status:             metav1.ConditionTrue,
			Reason:             "ResourcesAvailable",
			ObservedGeneration: generation,
			Message:            "All resources are available",
		}
	}
}

func (c *StatusSyncController) getSyncValues(obj *unstructured.Unstructured,
	statusSyncRules []workv1alpha1.StatusSyncRule) ([]workv1alpha1.SyncValue, metav1.Condition) {
	errs := []error{}
	values := []workv1alpha1.SyncValue{}

	for _, rule := range statusSyncRules {
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
			Type:    statusSyncConditionType,
			Reason:  "StatusSyncFailed",
			Status:  metav1.ConditionFalse,
			Message: fmt.Sprintf("Sync status failed with error %v", err),
		}
	}

	if len(values) == 0 {
		return values, metav1.Condition{
			Type:   statusSyncConditionType,
			Reason: "NoStatusSynced",
			Status: metav1.ConditionTrue,
		}
	}

	return values, metav1.Condition{
		Type:   statusSyncConditionType,
		Reason: "StatusSynced",
		Status: metav1.ConditionTrue,
	}
}

// buildAvailableStatusCondition returns a StatusCondition with type Available for a given manifest resource
func (c *StatusSyncController) buildAvailableStatusCondition(manifest workv1alpha1.Manifest) (
	*unstructured.Unstructured, metav1.Condition, error) {

	gvr, unstructuredObj, err := decodeUnstructured(manifest, c.restMapper)
	if err != nil {
		return nil, metav1.Condition{
			Type:    resourceAvailableConditionType,
			Status:  metav1.ConditionUnknown,
			Reason:  "IncompletedResourceMeta",
			Message: "Resource meta is incompleted",
		}, err
	}

	obj, err := c.spokeDynamicClient.Resource(gvr).Namespace(unstructuredObj.GetNamespace()).
		Get(context.TODO(), unstructuredObj.GetName(), metav1.GetOptions{})

	switch {
	case errors.IsNotFound(err):
		return nil, metav1.Condition{
			Type:    resourceAvailableConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  "ResourceNotAvailable",
			Message: "Resource is not available",
		}, err
	case err != nil:
		return nil, metav1.Condition{
			Type:    resourceAvailableConditionType,
			Status:  metav1.ConditionUnknown,
			Reason:  "FetchingResourceFailed",
			Message: fmt.Sprintf("Failed to fetch resource: %v", err),
		}, err
	}

	return obj, metav1.Condition{
		Type:    resourceAvailableConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  "ResourceAvailable",
		Message: "Resource is available",
	}, nil
}
