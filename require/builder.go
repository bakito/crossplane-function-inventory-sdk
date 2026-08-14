package require

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func As(objectName string) Builder {
	return &requirement{
		objectName: objectName,
	}
}

type requirement struct {
	objectName  string
	apiVersion  string
	kind        string
	ro          runtime.Object
	version     string
	matchName   string
	matchLabels map[string]string
	scheme      *runtime.Scheme
	namespace   *string
}

type Builder interface {
	WithGVK(apiVersion, kind string) RequirementBuilder
	WithRO(object runtime.Object) ROBuilder
}

func (r *requirement) WithGVK(apiVersion, kind string) RequirementBuilder {
	r.apiVersion = apiVersion
	r.kind = kind
	return r
}

func (r *requirement) WithRO(ro runtime.Object) ROBuilder {
	r.ro = ro
	return r
}

func (r *requirement) WithVersion(version string) ROBuilder {
	r.version = version
	return r
}

func (r *requirement) WithScheme(scheme runtime.Scheme) ROBuilder {
	r.scheme = &scheme
	return r
}

func (r *requirement) WithNamespace(namespace *string) ROBuilder {
	r.namespace = namespace
	return r
}

type ROBuilder interface {
	RequirementBuilder
	WithVersion(version string) ROBuilder
	WithScheme(scheme runtime.Scheme) ROBuilder
	WithNamespace(namespace *string) ROBuilder
}

type RequirementBuilder interface {
	MatchName(resourceName string) Requirement
	MatchLabels(labels map[string]string) Requirement
}

func (r *requirement) MatchName(resourceName string) Requirement {
	r.matchName = resourceName
	return r
}

func (r *requirement) MatchLabels(labels map[string]string) Requirement {
	r.matchLabels = labels
	return r
}
