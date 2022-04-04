package statusfeedback

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/jsonpath"
	workv1alpha1 "sigs.k8s.io/work-api/pkg/apis/v1alpha1"
	"sigs.k8s.io/work-api/pkg/statusfeedback/rules"
)

type StatusReader struct {
	CommonFieldsStatus rules.CommonFieldsStatusRuleResolver
}

func NewStatusReader() *StatusReader {
	return &StatusReader{
		CommonFieldsStatus: rules.DefaultCommonFieldsStatusRule(),
	}
}

func (s *StatusReader) GetValuesByRule(obj *unstructured.Unstructured, rule workv1alpha1.StatusFeedbackRule) ([]workv1alpha1.FeedbackValue, error) {
	errs := []error{}
	values := []workv1alpha1.FeedbackValue{}

	switch rule.Type {
	case workv1alpha1.CommonFieldsType:
		paths := s.CommonFieldsStatus.GetPathsByKind(obj.GroupVersionKind())
		if len(paths) == 0 {
			return values, fmt.Errorf("cannot find the CommonFields statuses for resrouce with gvk %s", obj.GroupVersionKind().String())
		}

		for _, path := range paths {
			value, err := getValueByJsonPath(path.Name, path.Path, obj)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if value == nil {
				continue
			}
			values = append(values, *value)
		}
	case workv1alpha1.JSONPathsType:
		for _, path := range rule.JsonPaths {
			// skip if version is specified and the object version does not match
			if len(path.Version) != 0 && obj.GroupVersionKind().Version != path.Version {
				errs = append(errs, fmt.Errorf("version set in the path %s is not matched for the related resource", path.Name))
				continue
			}

			value, err := getValueByJsonPath(path.Name, path.Path, obj)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if value == nil {
				continue
			}
			values = append(values, *value)
		}
	}

	return values, utilerrors.NewAggregate(errs)
}

func getValueByJsonPath(name, path string, obj *unstructured.Unstructured) (*workv1alpha1.FeedbackValue, error) {
	j := jsonpath.New(name).AllowMissingKeys(true)
	err := j.Parse(fmt.Sprintf("{%s}", path))
	if err != nil {
		return nil, fmt.Errorf("failed to parse json path %s of %s with error: %v", path, name, err)
	}

	results, err := j.FindResults(obj.UnstructuredContent())

	if err != nil {
		return nil, fmt.Errorf("failed to find value for %s with error: %v", name, err)
	}

	if len(results) == 0 || len(results[0]) == 0 {
		// no results are found here.
		return nil, nil
	}

	// as we only support simple JSON path, we can assume to have only one result (or none, filtered out above)
	value := results[0][0].Interface()

	if value == nil {
		// ignore the result if it is nil
		return nil, nil
	}

	var fieldValue workv1alpha1.FieldValue
	switch t := value.(type) {
	case int64:
		fieldValue = workv1alpha1.FieldValue{
			Type:    workv1alpha1.Integer,
			Integer: &t,
		}
		return &workv1alpha1.FeedbackValue{
			Name:  name,
			Value: fieldValue,
		}, nil
	case string:
		fieldValue = workv1alpha1.FieldValue{
			Type:   workv1alpha1.String,
			String: &t,
		}
		return &workv1alpha1.FeedbackValue{
			Name:  name,
			Value: fieldValue,
		}, nil
	case bool:
		fieldValue = workv1alpha1.FieldValue{
			Type:    workv1alpha1.Boolean,
			Boolean: &t,
		}
		return &workv1alpha1.FeedbackValue{
			Name:  name,
			Value: fieldValue,
		}, nil
	}

	return nil, fmt.Errorf("the type %v of the value for %s is not found", reflect.TypeOf(value), name)
}
