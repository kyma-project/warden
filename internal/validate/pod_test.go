package validate_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestValidatePod(t *testing.T) {
	testNs := "test-namespace"

	validImage := "validImage"
	validContainer := v1.Container{Name: "valid-image", Image: validImage}
	invalidImage := "invalidImage"
	invalidContainer := v1.Container{Name: "invalid-image", Image: invalidImage}
	longResp := "long"
	longRespContainer := v1.Container{Name: "invalid-image", Image: longResp}

	t.Run("Namespace mismatch with Pod Namespace", func(t *testing.T) {
		//GIVEN
		ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs}}
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "different"},
		}
		podValidator := validate.NewPodValidator(nil)

		//WHEN
		_, err := podValidator.ValidatePod(context.TODO(), pod, ns)
		//THEN
		require.Error(t, err)
		require.ErrorContains(t, err, "namespace mismatch")

	})

	testCases := []struct {
		name           string
		pod            *v1.Pod
		expectedResult validate.ValidationResult
	}{
		{
			name: "pod has valid image",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					validContainer,
				}}},
			expectedResult: validate.Valid,
		},
		{
			name: "pod has valid images",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					validContainer, validContainer,
				}}},
			expectedResult: validate.Valid,
		},
		{
			name: "pod has invalid image",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					invalidContainer}}},
			expectedResult: validate.Invalid,
		},
		{
			name: "image validator timeout",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					longRespContainer}}},
			expectedResult: validate.ServiceUnavailable,
		},
		{
			name: "pod has invalid image among otters",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					validContainer, validContainer, validContainer, invalidContainer,
				}}},
			expectedResult: validate.Invalid,
		},
		{
			name: "pod has valid image in initContainers and containers",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{validContainer},
					Containers:     []v1.Container{validContainer},
				}},
			expectedResult: validate.Valid,
		},
		{
			name: "pod has invalid image in initContainers",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{invalidContainer},
					Containers:     []v1.Container{validContainer},
				}},
			expectedResult: validate.Invalid,
		},
		{
			name: "pod has invalid image among others images in initContainers",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{validContainer, validContainer, invalidContainer},
					Containers:     []v1.Container{validContainer},
				}},
			expectedResult: validate.Invalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s-%s", "successfull", testCase.name), func(t *testing.T) {
			//GIVEN
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs,
				Labels: map[string]string{
					pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
				}}}

			mockValidator := mocks.ImageValidatorService{}
			mockValidator.Mock.On("Validate", mock.Anything, invalidImage).Return(errors.New("Invalid image"))
			mockValidator.Mock.On("Validate", mock.Anything, validImage).Return(nil)
			mockValidator.Mock.On("Validate", mock.Anything, longResp).Return(pkg.NewUnknownResultErr(nil))

			podValidator := validate.NewPodValidator(&mockValidator)
			//WHEN
			result, err := podValidator.ValidatePod(context.TODO(), testCase.pod, ns)

			//THEN
			require.NoError(t, err)
			require.Equal(t, testCase.expectedResult, result)
		})
	}
}
