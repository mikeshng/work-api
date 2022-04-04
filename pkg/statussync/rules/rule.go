package rules

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	workv1alpha1 "sigs.k8s.io/work-api/pkg/apis/v1alpha1"
)

type CommonFieldsStatusRuleResolver interface {
	GetPathsByKind(schema.GroupVersionKind) []workv1alpha1.JsonPath
}

type DefaultCommonFieldsStatusResolver struct {
	rules map[schema.GroupVersionKind][]workv1alpha1.JsonPath
}

var deploymentRule = []workv1alpha1.JsonPath{
	{
		Name: "ReadyReplicas",
		Path: ".status.readyReplicas",
	},
	{
		Name: "Replicas",
		Path: ".status.replicas",
	},
	{
		Name: "AvailableReplicas",
		Path: ".status.availableReplicas",
	},
}

var jobRule = []workv1alpha1.JsonPath{
	{
		Name: "JobComplete",
		Path: `.status.conditions[?(@.type=="Complete")].status`,
	},
	{
		Name: "JobSucceeded",
		Path: `.status.succeeded`,
	},
}

var podRule = []workv1alpha1.JsonPath{
	{
		Name: "PodReady",
		Path: `.status.conditions[?(@.type=="Ready")].status`,
	},
	{
		Name: "PodPhase",
		Path: `.status.phase`,
	},
}

func DefaultCommonFieldsStatusRule() CommonFieldsStatusRuleResolver {
	return &DefaultCommonFieldsStatusResolver{
		rules: map[schema.GroupVersionKind][]workv1alpha1.JsonPath{
			{Group: "apps", Version: "v1", Kind: "Deployment"}: deploymentRule,
			{Group: "batch", Version: "v1", Kind: "Job"}:       jobRule,
			{Group: "", Version: "v1", Kind: "Pod"}:            podRule,
		},
	}
}

func (w *DefaultCommonFieldsStatusResolver) GetPathsByKind(gvk schema.GroupVersionKind) []workv1alpha1.JsonPath {
	return w.rules[gvk]
}
