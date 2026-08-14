package tag_const_gen

import (
	corev1 "k8s.io/api/core/v1"
)

type Observed struct {
	ObservedComposed *corev1.Namespace `crossplane:"observed-composed:ns-test"`
}
type Desired struct {
	DesiredComposed *corev1.ConfigMap `crossplane:"desired-composed:cm-test"`
}
