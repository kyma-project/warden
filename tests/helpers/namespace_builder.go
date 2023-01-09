package helpers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceBuilder struct {
	tc        *testContext
	namespace *corev1.Namespace
}

func (tc *testContext) Namespace() *NamespaceBuilder {
	return &NamespaceBuilder{
		tc: tc,
		namespace: &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: tc.UniqueName(),
			},
		},
	}
}

func (b *NamespaceBuilder) WithName(name string) *NamespaceBuilder {
	b.namespace.Name = name
	return b
}

func (b *NamespaceBuilder) WithValidation(enabled bool) *NamespaceBuilder {
	if !enabled {
		return b
	}
	if b.namespace.ObjectMeta.Labels == nil {
		b.namespace.ObjectMeta.Labels = map[string]string{}
	}
	b.namespace.ObjectMeta.Labels["namespaces.warden.kyma-project.io/validate"] = "enabled"
	return b
}

func (b *NamespaceBuilder) Build() *corev1.Namespace {
	return b.namespace
}
