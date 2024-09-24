package validate_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/validate/mocks"
	"github.com/kyma-project/warden/pkg"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidatePod(t *testing.T) {
	testNs := "test-namespace"

	validImage := "validImage"
	validContainer := v1.Container{Name: "valid-image", Image: validImage}
	invalidImage := "invalidImage"
	invalidContainer := v1.Container{Name: "invalid-image", Image: invalidImage}
	invalidImage2 := "invalidImage2"
	invalidContainer2 := v1.Container{Name: "invalid-image2", Image: invalidImage2}
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
		name                 string
		pod                  *v1.Pod
		expectedStatus       validate.ValidationStatus
		expectedFailedImages []string
	}{
		{
			name: "pod has valid image",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					validContainer,
				}}},
			expectedStatus:       validate.Valid,
			expectedFailedImages: []string{},
		},
		{
			name: "pod has valid images",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					validContainer, validContainer,
				}}},
			expectedStatus:       validate.Valid,
			expectedFailedImages: []string{},
		},
		{
			name: "pod has invalid image",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					invalidContainer}}},
			expectedStatus:       validate.Invalid,
			expectedFailedImages: []string{invalidImage},
		},
		{
			name: "pod has two invalid images",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					invalidContainer,
					invalidContainer2}}},
			expectedStatus:       validate.Invalid,
			expectedFailedImages: []string{invalidImage, invalidImage2},
		},
		{
			name: "pod has one invalid image",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					validContainer,
					invalidContainer}}},
			expectedStatus:       validate.Invalid,
			expectedFailedImages: []string{invalidImage},
		},
		{
			name: "image validator timeout",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					longRespContainer}}},
			expectedStatus:       validate.ServiceUnavailable,
			expectedFailedImages: []string{longResp},
		},
		{
			name: "pod has invalid image among otters",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{Containers: []v1.Container{
					validContainer, validContainer, validContainer, invalidContainer,
				}}},
			expectedStatus:       validate.Invalid,
			expectedFailedImages: []string{invalidImage},
		},
		{
			name: "pod has valid image in initContainers and containers",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{validContainer},
					Containers:     []v1.Container{validContainer},
				}},
			expectedStatus:       validate.Valid,
			expectedFailedImages: []string{},
		},
		{
			name: "pod has invalid image in initContainers",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{invalidContainer},
					Containers:     []v1.Container{validContainer},
				}},
			expectedStatus:       validate.Invalid,
			expectedFailedImages: []string{invalidImage},
		},
		{
			name: "pod has invalid image among others images in initContainers",
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNs},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{validContainer, validContainer, invalidContainer},
					Containers:     []v1.Container{validContainer},
				}},
			expectedStatus:       validate.Invalid,
			expectedFailedImages: []string{invalidImage},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s-%s", "successfull", testCase.name), func(t *testing.T) {
			//GIVEN
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNs,
				Labels: map[string]string{
					pkg.NamespaceValidationLabel: pkg.NamespaceValidationEnabled,
				}}}

			validatorSvcMock := mocks.ImageValidatorService{}
			validatorSvcMock.Mock.On("Validate", mock.Anything, invalidImage).Return(errors.New("Invalid image"))
			validatorSvcMock.Mock.On("Validate", mock.Anything, invalidImage2).Return(errors.New("Invalid image"))
			validatorSvcMock.Mock.On("Validate", mock.Anything, validImage).Return(nil)
			validatorSvcMock.Mock.On("Validate", mock.Anything, longResp).Return(pkg.NewUnknownResultErr(nil))

			podValidator := validate.NewPodValidator(&validatorSvcMock)
			//WHEN
			result, err := podValidator.ValidatePod(context.TODO(), testCase.pod, ns)

			//THEN
			require.NoError(t, err)
			require.Equal(t, testCase.expectedStatus, result.Status)
			require.ElementsMatchf(t, testCase.expectedFailedImages, result.InvalidImages, "list of images do not match")
		})
	}
}

func TestNewValidatorSvc(t *testing.T) {
	t.Run("create new validator svc", func(t *testing.T) {
		validatorSvc := validate.NewValidatorSvcFactory().
			NewValidatorSvc("notaryURL", "allowed,registries", time.Second)
		result, err := validatorSvc.ValidatePod(context.Background(), &v1.Pod{}, &v1.Namespace{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, validate.Valid, result.Status)
	})
}
