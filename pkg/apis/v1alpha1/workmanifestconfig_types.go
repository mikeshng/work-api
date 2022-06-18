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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WorkManifestConfig represents the configuration of a workload manifest.
// This resource allows the user to customize the configuration of a manifest inside a Work object.
// WorkManifestConfig is a cluster-scoped resource.
type WorkManifestConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the configuration for a workload manifest.
	// +required
	Spec WorkManifestConfigSpec `json:"spec"`
}

// WorkManifestConfigSpec provides information for the WorkManifestConfig
type WorkManifestConfigSpec struct {
	// ManifestGVK represents the type of manifest this configuration is for
	// +required
	ManifestGVK ManifestGVK `json:"manifestGVK,omitempty"`

	// ResourceStatusSyncConfiguration represents the configuration of workload manifest status sync.
	// +optional
	ResourceStatusSyncConfig ResourceStatusSyncConfiguration `json:"resourceStatusSync,omitempty"`
}

// ManifestGVK represents the type of manifest this configuration is for.
type ManifestGVK struct {
	// Group is the group of the workload resource manifest.
	Group string `json:"group"`

	// Version is the version of the workload resource manifest.
	Version string `json:"version"`

	// Kind is the kind of the workload resource manifest.
	Kind string `json:"kind"`
}

// ResourceStatusSyncConfiguration represents the resource status sync configuration of a workload manifest.
type ResourceStatusSyncConfiguration struct {
	// StatusSyncRule defines what resource status field should be returned.
	// +optional
	Rules []StatusSyncRule `json:"rules"`

	// FrequencySeconds represents how often (in seconds) to perform the probe.
	// Default to 60 seconds. Minimum value is 1.
	// +optional
	FrequencySeconds int32 `json:"frequencySeconds,omitempty"`

	// StopSyncThreshold represents minimum consecutive probe before stopping the sync.
	// Defaults to 0. Minimum value is 0. The value 0 represents never stop syncing.
	// +optional
	StopSyncThreshold int32 `json:"stopThreshold,omitempty"`
}

// StatusSyncRule represents a resource status field should be returned.
type StatusSyncRule struct {
	// Type defines the option of how status can be returned.
	// It can be JSONPaths or Scripts.
	// If the type is JSONPaths, user should specify the jsonPaths field.
	// If the type is Scripts, user should specify the scripts field.
	// +kubebuilder:validation:Required
	// +required
	Type SyncType `json:"type"`

	// JsonPaths defines the json path under status field to be synced.
	// +optional
	JsonPaths []JsonPath `json:"jsonPaths,omitempty"`

	// Scripts defines the script evaluation under status field to be synced.
	// +optional
	Scripts []Script `json:"scripts,omitempty"`
}

// SyncType represents the option of how status can be returned.
// +kubebuilder:validation:Enum=JSONPaths;Scripts
type SyncType string

const (
	// JSONPathsType represents that values of status fields with certain json paths specified will be
	// returned
	JSONPathsType SyncType = "JSONPaths"

	// ScriptsType represents that values of status fields with certain scripts specified will be
	// returned
	ScriptsType SyncType = "Scripts"
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
	// of 'StatusSynced' will be set as false.
	// Ref to https://kubernetes.io/docs/reference/kubectl/jsonpath/ on how to write a jsonPath.
	// +kubebuilder:validation:Required
	// +required
	Path string `json:"path"`
}

// Script represents a status field to be synced for a manifest using a scripting language evaluation.
type Script struct {
	// Name represents the alias name for this field
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`

	// Language represents the language of the script.
	// +kubebuilder:validation:Required
	// +required
	Language string `json:"language"`

	// Content represents the script that will be evaluated against the workload resource status field.
	// +kubebuilder:validation:Required
	// +required
	Content string `json:"content"`
}

// +kubebuilder:object:root=true

// WorkManifestConfigList contains a list of WorkManifestConfigs
type WorkManifestConfigList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of workManifestConfigs.
	// +listType=set
	Items []WorkManifestConfig `json:"items"`
}
