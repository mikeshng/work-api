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

// WorkSpec defines the desired state of Work
type WorkSpec struct {
	// Workload represents the manifest workload to be deployed on spoke cluster
	Workload WorkloadTemplate `json:"workload,omitempty"`
}

// WorkloadTemplate represents the manifest workload to be deployed on spoke cluster
type WorkloadTemplate struct {
	// Manifests represents a list of kuberenetes resources to be deployed on the spoke cluster.
	// +optional
	Manifests []Manifest `json:"manifests,omitempty"`
}

// Manifest represents a resource to be deployed on spoke cluster
type Manifest struct {
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	runtime.RawExtension `json:",inline"`

	// StatusFeedbackRules defines the resource status fields should be returned.
	// +optional
	StatusFeedbackRules []StatusFeedbackRule `json:"statusFeedbackRules"`
}

// StatusFeedbackRule represents a resource status field should be returned.
type StatusFeedbackRule struct {
	// Type defines the option of how status can be returned.
	// It can be JSONPaths or CommonFields.
	// If the type is JSONPaths, user should specify the jsonPaths field
	// If the type is CommonFields, certain common fields of status defined by a rule only
	// for types in in k8s.io/api will be reported,
	// If these status fields do not exist, no values will be reported.
	// +kubebuilder:validation:Required
	// +required
	Type FeedBackType `json:"type"`

	// JsonPaths defines the json path under status field to be synced.
	// +optional
	JsonPaths []JsonPath `json:"jsonPaths,omitempty"`
}

// FeedBackType represents the option of how status can be returned.
// +kubebuilder:validation:Enum=CommonFields;JSONPaths
type FeedBackType string

const (
	// CommonFieldsType represents that values of some common status fields will be returned, which
	// is reflected with a hardcoded rule only for types in k8s.io/api.
	CommonFieldsType FeedBackType = "CommonFields"

	// JSONPathsType represents that values of status fields with certain json paths specified will be
	// returned
	JSONPathsType FeedBackType = "JSONPaths"
)

// JsonPath represents a status field to be synced for a manifest using json path.
type JsonPath struct {
	// Name represents the alias name for this field
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`

	// Version is the version of the Kubernetes resource.
	// If it is not specified, the resource with the semantically latest version is
	// used to resolve the path.
	// +optional
	Version string `json:"version,omitempty"`

	// Path represents the json path of the field under status.
	// The path must point to a field with single value in the type of integer, bool or string.
	// If the path points to a non-existing field, no value will be returned.
	// If the path points to a structure, map or slice, no value will be returned and the status conddition
	// of 'StatusFeedBackSynced' will be set as false.
	// Ref to https://kubernetes.io/docs/reference/kubectl/jsonpath/ on how to write a jsonPath.
	// +kubebuilder:validation:Required
	// +required
	Path string `json:"path"`
}

// WorkStatus defines the observed state of Work
type WorkStatus struct {
	// Conditions contains the different condition statuses for this work.
	// Valid condition types are:
	// 1. Applied represents workload in Work is applied successfully on the spoke cluster.
	// 2. Progressing represents workload in Work in the trasitioning from one state to another the on the spoke cluster.
	// 3. Available represents workload in Work exists on the spoke cluster.
	// 4. Degraded represents the current state of workload does not match the desired
	// state for a certain period.
	Conditions []metav1.Condition `json:"conditions"`

	// ManifestConditions represents the conditions of each resource in work deployed on
	// spoke cluster.
	// +optional
	ManifestConditions []ManifestCondition `json:"manifestConditions,omitempty"`
}

// ResourceIdentifier provides the identifiers needed to interact with any arbitrary object.
type ResourceIdentifier struct {
	// Ordinal represents an index in manifests list, so the condition can still be linked
	// to a manifest even thougth manifest cannot be parsed successfully.
	Ordinal int `json:"ordinal,omitempty"`

	// Group is the group of the resource.
	Group string `json:"group,omitempty"`

	// Version is the version of the resource.
	Version string `json:"version,omitempty"`

	// Kind is the kind of the resource.
	Kind string `json:"kind,omitempty"`

	// Resource is the resource type of the resource
	Resource string `json:"resource,omitempty"`

	// Namespace is the namespace of the resource, the resource is cluster scoped if the value
	// is empty
	Namespace string `json:"namespace,omitempty"`

	// Name is the name of the resource
	Name string `json:"name,omitempty"`
}

// ManifestCondition represents the conditions of the resources deployed on
// spoke cluster
type ManifestCondition struct {
	// resourceId represents a identity of a resource linking to manifests in spec.
	// +required
	Identifier ResourceIdentifier `json:"identifier,omitempty"`

	// Conditions represents the conditions of this resource on spoke cluster
	// +required
	Conditions []metav1.Condition `json:"conditions"`

	// StatusFeedbacks represents the values of the field synced back defined in statusFeedbackRules
	// +optional
	StatusFeedbacks StatusFeedbackResult `json:"statusFeedback,omitempty"`
}

// StatusFeedbackResult represents the values of the field synced back defined in statusFeedbackRules
type StatusFeedbackResult struct {
	// Values represents the synced value of the interested field.
	// +listType:=map
	// +listMapKey:=name
	// +optional
	Values []FeedbackValue `json:"values,omitempty"`
}

// FeedbackValue represents the synced value of the feedback field.
type FeedbackValue struct {
	// Name represents the alias name for this field. It is the same as what is specified
	// in StatusFeedbackRule in the spec.
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`

	// Value is the value of the status field.
	// The value of the status field can only be integer, string or boolean.
	// +kubebuilder:validation:Required
	// +required
	Value FieldValue `json:"fieldValue"`
}

// FieldValues represents the value of the field
// The value of the status field can only be integer, string or boolean.
type FieldValue struct {
	// Type represents the type of the value, it can be integer, string or boolean.
	// +kubebuilder:validation:Required
	// +required
	Type ValueType `json:"type"`

	// Integer is the integer value when type is integer.
	// +optional
	Integer *int64 `json:"integer,omitempty"`

	// String is the string value when when type is string.
	// +optional
	String *string `json:"string,omitempty"`

	// Boolean is bool value when type is boolean.
	// +optional
	Boolean *bool `json:"boolean,omitempty"`
}

// Type represents the type of the value, it can by integer, string or bool
// +kubebuilder:validation:Enum=Integer;String;Boolean
type ValueType string

const (
	Integer ValueType = "Integer"
	String  ValueType = "String"
	Boolean ValueType = "Boolean"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Work is the Schema for the works API
type Work struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the workload of a work.
	// +optional
	Spec WorkSpec `json:"spec,omitempty"`
	// status defines the status of each applied manifest on the spoke cluster.
	Status WorkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkList contains a list of Work
type WorkList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of works.
	// +listType=set
	Items []Work `json:"items"`
}
