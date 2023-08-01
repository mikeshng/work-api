/*
Copyright 2021 The Kubernetes Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// Work represents a manifests workload that hub wants to deploy on the managed cluster.
// A manifest workload is defined as a set of Kubernetes resources.
// Work must be created in the cluster namespace on the hub, so that agent on the
// corresponding managed cluster can access this resource and deploy on the managed
// cluster.
type Work struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents a desired configuration of work to be deployed on the managed cluster.
	Spec WorkSpec `json:"spec"`

	// Status represents the current status of work.
	// +optional
	Status WorkStatus `json:"status,omitempty"`
}

// WorkSpec represents a desired configuration of manifests to be deployed on the managed cluster.
type WorkSpec struct {
	// Workload represents the manifest workload to be deployed on a managed cluster.
	Workload ManifestsTemplate `json:"workload,omitempty"`
}

// Manifest represents a resource to be deployed on managed cluster.
type Manifest struct {
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	runtime.RawExtension `json:",inline"`
}

// ManifestsTemplate represents the manifest workload to be deployed on a managed cluster.
type ManifestsTemplate struct {
	// Manifests represents a list of kuberenetes resources to be deployed on a managed cluster.
	// +optional
	Manifests []Manifest `json:"manifests,omitempty"`
}

// ResourceIdentifier identifies a single resource included in this work
type ResourceIdentifier struct {
	// Group is the API Group of the Kubernetes resource,
	// empty string indicates it is in core group.
	// +optional
	Group string `json:"group"`

	// Resource is the resource name of the Kubernetes resource.
	// +kubebuilder:validation:Required
	// +required
	Resource string `json:"resource"`

	// Name is the name of the Kubernetes resource.
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`

	// Name is the namespace of the Kubernetes resource, empty string indicates
	// it is a cluster scoped resource.
	// +optional
	Namespace string `json:"namespace"`
}

// ManifestResourceMeta represents the group, version, kind, as well as the group, version, resource, name and namespace of a resoure.
type ManifestResourceMeta struct {
	// Ordinal represents the index of the manifest on spec.
	// +required
	Ordinal int32 `json:"ordinal"`

	// Group is the API Group of the Kubernetes resource.
	// +optional
	Group string `json:"group"`

	// Version is the version of the Kubernetes resource.
	// +optional
	Version string `json:"version"`

	// Kind is the kind of the Kubernetes resource.
	// +optional
	Kind string `json:"kind"`

	// Resource is the resource name of the Kubernetes resource.
	// +optional
	Resource string `json:"resource"`

	// Name is the name of the Kubernetes resource.
	// +optional
	Name string `json:"name"`

	// Name is the namespace of the Kubernetes resource.
	// +optional
	Namespace string `json:"namespace"`
}

// AppliedManifestResourceMeta represents the group, version, resource, name and namespace of a resource.
// Since these resources have been created, they must have valid group, version, resource, namespace, and name.
type AppliedManifestResourceMeta struct {
	ResourceIdentifier `json:",inline"`

	// Version is the version of the Kubernetes resource.
	// +kubebuilder:validation:Required
	// +required
	Version string `json:"version"`

	// UID is set on successful deletion of the Kubernetes resource by controller. The
	// resource might be still visible on the managed cluster after this field is set.
	// It is not directly settable by a client.
	// +optional
	UID string `json:"uid,omitempty"`
}

// WorkStatus represents the current status of managed cluster Work.
type WorkStatus struct {
	// Conditions contains the different condition statuses for this work.
	// Valid condition types are:
	// 1. Applied represents workload in Work is applied successfully on managed cluster.
	// 2. Progressing represents workload in Work is being applied on managed cluster.
	// 3. Available represents workload in Work exists on the managed cluster.
	// 4. Degraded represents the current state of workload does not match the desired
	// state for a certain period.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ResourceStatus represents the status of each resource in work deployed on a
	// managed cluster. The Klusterlet agent on managed cluster syncs the condition from the managed cluster to the hub.
	// +optional
	ResourceStatus ManifestResourceStatus `json:"resourceStatus,omitempty"`
}

// ManifestResourceStatus represents the status of each resource in manifest work deployed on
// managed cluster
type ManifestResourceStatus struct {
	// Manifests represents the condition of manifests deployed on managed cluster.
	// Valid condition types are:
	// 1. Progressing represents the resource is being applied on managed cluster.
	// 2. Applied represents the resource is applied successfully on managed cluster.
	// 3. Available represents the resource exists on the managed cluster.
	// 4. Degraded represents the current state of resource does not match the desired
	// state for a certain period.
	Manifests []ManifestCondition `json:"manifests,omitempty"`
}

const (
	// WorkProgressing represents that the work is in the progress to be
	// applied on the managed cluster.
	WorkProgressing string = "Progressing"
	// WorkApplied represents that the workload defined in work is
	// succesfully applied on the managed cluster.
	WorkApplied string = "Applied"
	// WorkAvailable represents that all resources of the work exists on
	// the managed cluster.
	WorkAvailable string = "Available"
	// WorkDegraded represents that the current state of work does not match
	// the desired state for a certain period.
	WorkDegraded string = "Degraded"
)

// ManifestCondition represents the conditions of the resources deployed on a
// managed cluster.
type ManifestCondition struct {
	// ResourceMeta represents the group, version, kind, name and namespace of a resoure.
	// +required
	ResourceMeta ManifestResourceMeta `json:"resourceMeta"`

	// Conditions represents the conditions of this resource on a managed cluster.
	// +required
	Conditions []metav1.Condition `json:"conditions"`
}

// ManifestConditionType represents the condition type of a single
// resource manifest deployed on the managed cluster.
type ManifestConditionType string

const (
	// ManifestProgressing represents that the resource is being applied on the managed cluster
	ManifestProgressing ManifestConditionType = "Progressing"
	// ManifestApplied represents that the resource object is applied
	// on the managed cluster.
	ManifestApplied ManifestConditionType = "Applied"
	// ManifestAvailable represents that the resource object exists
	// on the managed cluster.
	ManifestAvailable ManifestConditionType = "Available"
	// ManifestDegraded represents that the current state of resource object does not
	// match the desired state for a certain period.
	ManifestDegraded ManifestConditionType = "Degraded"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkList is a collection of works.
type WorkList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of works.
	Items []Work `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppliedWork represents an applied work on managed cluster that is placed
// on a managed cluster. An AppliedWork links to a work on a hub recording resources
// deployed in the managed cluster.
// When the agent is removed from managed cluster, cluster-admin on managed cluster
// can delete appliedwork to remove resources deployed by the agent.
// The name of the appliedwork must be in the format of
// {hash of hub's first kube-apiserver url}-{work name}
type AppliedWork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired configuration of AppliedWork.
	Spec AppliedWorkSpec `json:"spec,omitempty"`

	// Status represents the current status of AppliedWork.
	// +optional
	Status AppliedWorkStatus `json:"status,omitempty"`
}

// AppliedWorkSpec represents the desired configuration of AppliedWork
type AppliedWorkSpec struct {
	// HubHash represents the hash of the first hub kube apiserver to identify which hub
	// this AppliedWork links to.
	// +required
	HubHash string `json:"hubHash"`

	// AgentID represents the ID of the work agent who is to handle this AppliedWork.
	AgentID string `json:"agentID"`

	// WorkName represents the name of the related work on the hub.
	// +required
	WorkName string `json:"workName"`
}

// AppliedWorkStatus represents the current status of AppliedWork
type AppliedWorkStatus struct {
	// AppliedResources represents a list of resources defined within the work that are applied.
	// Only resources with valid GroupVersionResource, namespace, and name are suitable.
	// An item in this slice is deleted when there is no mapped manifest in work.Spec or by finalizer.
	// The resource relating to the item will also be removed from managed cluster.
	// The deleted resource may still be present until the finalizers for that resource are finished.
	// However, the resource will not be undeleted, so it can be removed from this list and eventual consistency is preserved.
	// +optional
	AppliedResources []AppliedManifestResourceMeta `json:"appliedResources,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppliedWorkList is a collection of appliedworks.
type AppliedWorkList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of appliedworks.
	Items []AppliedWork `json:"items"`
}
