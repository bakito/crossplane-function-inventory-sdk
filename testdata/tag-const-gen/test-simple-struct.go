package tag_const_gen

import (
	"github.com/bakito/crossplane-function-inventory-sdk/inventory"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	corev1 "k8s.io/api/core/v1"
)

type SimpleStruct struct {
	ObservedComposed      *corev1.Namespace           `crossplane:"observed-composed:ns-test"`
	ObservedComposedReady fnv1.Ready                  `crossplane:"observed-composed:ns-test"`
	ObservedComposedConn  inventory.ConnectionDetails `crossplane:"observed-composed:ns-test"`

	DesiredComposed      *corev1.ConfigMap           `crossplane:"desired-composed:cm-test"`
	DesiredComposedReady fnv1.Ready                  `crossplane:"desired-composed:cm-test"`
	DesiredComposedConn  inventory.ConnectionDetails `crossplane:"desired-composed:cm-test"`

	ObservedComposite      *corev1.Secret              `crossplane:"observed-composite"`
	ObservedCompositeReady fnv1.Ready                  `crossplane:"observed-composite"`
	ObservedCompositeConn  inventory.ConnectionDetails `crossplane:"observed-composite"`

	DesiredComposite      *corev1.Secret              `crossplane:"desired-composite"`
	DesiredCompositeReady fnv1.Ready                  `crossplane:"desired-composite"`
	DesiredCompositeConn  inventory.ConnectionDetails `crossplane:"desired-composite"`

	RequiredPod      *corev1.Pod `crossplane:"required:pod-required"`
	RequiredPodReady fnv1.Ready  `crossplane:"required:pod-required"`

	RequiredSvc      []*corev1.Service `crossplane:"required:svc-required"`
	RequiredSvcReady []fnv1.Ready      `crossplane:"required:svc-required"`
}
