package require

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

type Requirement interface {
	Get() (name string, selector *fnv1.ResourceSelector, err error)
}

func (r *requirement) Get() (name string, selector *fnv1.ResourceSelector, err error) {
	sel := &fnv1.ResourceSelector{}
	if r.ro == nil {
		if r.apiVersion == "" {
			return "", nil, errors.New("apiVersion is required")
		}
		if r.kind == "" {
			return "", nil, errors.New("kind is required")
		}
		sel.ApiVersion = r.apiVersion
		sel.Kind = r.kind
	} else {
		gvk, err := findGVK(r.ro, r.scheme, r.version)
		if err != nil {
			return "", nil, err
		}
		sel.ApiVersion = gvk.GroupVersion().String()
		sel.Kind = gvk.Kind
		if r.namespace != nil {
			sel.Namespace = r.namespace
		}
	}

	if len(r.matchLabels) > 0 {
		sel.Match = &fnv1.ResourceSelector_MatchLabels{
			MatchLabels: &fnv1.MatchLabels{Labels: r.matchLabels},
		}
	} else {
		sel.Match = &fnv1.ResourceSelector_MatchName{
			MatchName: r.matchName,
		}
	}

	return r.objectName, sel, nil
}

func findGVK(
	object runtime.Object,
	customScheme *runtime.Scheme,
	version string,
) (schema.GroupVersionKind, error) {
	scheme := composed.Scheme
	if customScheme != nil {
		scheme = customScheme
	}

	var gvk schema.GroupVersionKind

	objectKinds, _, err := scheme.ObjectKinds(object)
	if err != nil {
		return gvk, err
	}

	if len(objectKinds) == 0 {
		return gvk, errors.New("no object kinds found")
	}

	if len(objectKinds) > 1 || version != "" {
		if version == "" {
			return gvk, errors.New("version required for multiple object kinds")
		}
		found := false
		for _, ok := range objectKinds {
			if ok.Version == version {
				gvk = ok
				found = true
				break
			}
		}
		if !found {
			return gvk, fmt.Errorf("no object kind found with version %s", version)
		}
	} else {
		gvk = objectKinds[0]
	}

	return gvk, err
}
