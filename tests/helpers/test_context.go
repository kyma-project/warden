package helpers

import (
	"context"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

type testContext struct {
	test              *testing.T
	client            ctrlclient.Client
	validationEnabled bool
	namePrefix        string
	namespaceName     string
	namespace         *corev1.Namespace
}

func NewTestContext(t *testing.T, namePrefix string) *testContext {
	tc := testContext{
		test:              t,
		validationEnabled: false,
		namePrefix:        namePrefix,
	}
	tc.namespaceName = tc.UniqueName()
	return &tc
}

func (tc *testContext) ValidationEnabled(enabled bool) *testContext {
	tc.validationEnabled = enabled
	return tc
}

func (tc *testContext) Initialize() *testContext {
	var err error
	tc.client, err = ctrlclient.New(ctrl.GetConfigOrDie(), ctrlclient.Options{})
	require.NoError(tc.test, err)
	tc.CreateNamespace()
	return tc
}

func (tc *testContext) Destroy() {
	if tc.namespace != nil {
		tc.DeleteNamespace()
	}
}

func (tc *testContext) UniqueName() string {
	return tc.namePrefix + "-" + uuid.New().String()
}

func (tc *testContext) CreateNamespace() {
	if tc.namespace != nil {
		panic("Unexpected existing namespace when create namespace!")
	}
	tc.namespace = tc.Namespace().WithName(tc.namespaceName).WithValidation(tc.validationEnabled).Build()
	err := tc.Create(tc.namespace)
	require.NoError(tc.test, err)
}

func (tc *testContext) DeleteNamespace() {
	err := tc.Delete(tc.namespace)
	require.NoError(tc.test, err)
}

func (tc *testContext) Create(obj ctrlclient.Object) error {
	err := tc.client.Create(context.TODO(), obj)
	return err
}

func (tc *testContext) Update(obj ctrlclient.Object) error {
	err := tc.client.Update(context.TODO(), obj)
	return err
}

func (tc *testContext) Get(src, dest ctrlclient.Object) error {
	key := ctrlclient.ObjectKeyFromObject(src)
	err := tc.client.Get(context.TODO(), key, dest)
	return err
}

func (tc *testContext) Delete(obj ctrlclient.Object) error {
	err := tc.client.Delete(context.TODO(), obj)
	return err
}

func (tc *testContext) GetPodWhenReady(src, dest *v1.Pod) error {
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		err := tc.Get(src, dest)
		if err != nil {
			return err
		}
		if dest.Status.Phase == v1.PodRunning {
			return nil
		}
	}
	return errors.New("Pod still is not ready!")
}

func (tc *testContext) GetPodWhenCondition(src, dest *v1.Pod, condition func(*v1.Pod) bool) error {
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		err := tc.Get(src, dest)
		if err != nil {
			return err
		}
		if condition(dest) {
			return nil
		}
	}
	return errors.New("Pod still is not ready!")
}
