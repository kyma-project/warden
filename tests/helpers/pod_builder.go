package helpers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodBuilder struct {
	tc  *testContext
	pod *corev1.Pod
}

func (tc *testContext) Pod() *PodBuilder {
	return &PodBuilder{
		tc: tc,
		pod: &corev1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      tc.NameWithTime(),
				Namespace: tc.namespaceName,
			},
		},
	}
}

func (b *PodBuilder) WithContainer(container corev1.Container) *PodBuilder {
	b.pod.Spec.Containers = append(b.pod.Spec.Containers, container)
	return b
}

func (b *PodBuilder) Build() *corev1.Pod {
	return b.pod
}
