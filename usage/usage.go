package usage

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	prot "github.com/crossplane/crossplane/apis/v2/protection/v1beta1"
)

type UsageObject struct {
	TypeMeta   *metav1.TypeMeta
	ObjectMeta *metav1.ObjectMeta
}

func CreateUsageObject(
	typeMeta *metav1.TypeMeta,
	objectMeta *metav1.ObjectMeta,
) UsageObject {
	return UsageObject{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
	}
}

type Resource struct {
	Of  *UsageObject
	By  *UsageObject
	Log logging.Logger
}

// Cluster returns a cluster scoped usage resource.
func (usage *Resource) Cluster() *prot.ClusterUsage {
	return &prot.ClusterUsage{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec: prot.ClusterUsageSpec{
			Of:             *createResource(usage.Of),
			By:             createResource(usage.By),
			ReplayDeletion: new(true),
		},
	}
}

// Namespaced returns a namespaced usage resource.
func (usage *Resource) Namespaced() *prot.Usage {
	return &prot.Usage{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec: prot.UsageSpec{
			Of:             createNamespacedResource(usage.Of),
			By:             createResource(usage.By),
			ReplayDeletion: new(true),
		},
	}
}

func createResource(object *UsageObject) *prot.Resource {
	return &prot.Resource{
		APIVersion:  object.TypeMeta.APIVersion,
		Kind:        object.TypeMeta.Kind,
		ResourceRef: &prot.ResourceRef{Name: object.ObjectMeta.Name},
	}
}

func createNamespacedResource(object *UsageObject) prot.NamespacedResource {
	return prot.NamespacedResource{
		APIVersion:  object.TypeMeta.APIVersion,
		Kind:        object.TypeMeta.Kind,
		ResourceRef: &prot.NamespacedResourceRef{Name: object.ObjectMeta.Name},
	}
}
