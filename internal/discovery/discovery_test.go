// Copyright (c) 2021 GPBR Participacoes LTDA.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package discovery_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sdisc "k8s.io/client-go/discovery"

	"github.com/Gympass/cdn-origin-controller/internal/discovery"
)

type discMock struct {
	mock.Mock
	k8sdisc.DiscoveryInterface
}

func (m *discMock) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	args := m.Called(groupVersion)
	listObj := args.Get(0)
	if listObj == nil {
		return nil, args.Error(1)
	}
	list := listObj.(**metav1.APIResourceList)
	return *list, args.Error(1)
}

var (
	ingressAvailableServerResources = &metav1.APIResourceList{
		APIResources: []metav1.APIResource{
			{
				Kind: "Ingress",
			},
		},
	}
	ingressNotAvailableServerResources = &metav1.APIResourceList{}
)

var (
	noError              error
	v1beta1AvailableDisc = func() k8sdisc.DiscoveryInterface {
		m := discMock{}
		m.On("ServerResourcesForGroupVersion", networkingv1beta1.SchemeGroupVersion.String()).Return(&ingressAvailableServerResources, noError)
		m.On("ServerResourcesForGroupVersion", mock.Anything).Return(&ingressNotAvailableServerResources, noError)
		return &m
	}()

	v1AvailableDisc = func() k8sdisc.DiscoveryInterface {
		m := discMock{}
		m.On("ServerResourcesForGroupVersion", networkingv1.SchemeGroupVersion.String()).Return(&ingressAvailableServerResources, noError)
		m.On("ServerResourcesForGroupVersion", mock.Anything).Return(&ingressNotAvailableServerResources, noError)
		return &m
	}()

	v1ErrOnServerResourcesDisc = func() k8sdisc.DiscoveryInterface {
		m := discMock{}
		m.On("ServerResourcesForGroupVersion", mock.Anything).Return(nil, errors.New("mock err"))
		return &m
	}()

	v1beta1ErrOnServerResourcesDisc = func() k8sdisc.DiscoveryInterface {
		m := discMock{}
		m.On("ServerResourcesForGroupVersion", mock.Anything).Return(nil, errors.New("mock err"))
		return &m
	}()
)

func TestRunDiscoveryTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &DiscoveryTestSuite{})
}

type DiscoveryTestSuite struct {
	suite.Suite
}

func (s *DiscoveryTestSuite) TestHasV1beta1Ingress() {
	testCases := []struct {
		name    string
		client  k8sdisc.DiscoveryInterface
		want    bool
		wantErr bool
	}{
		{
			"V1beta1 is available",
			v1beta1AvailableDisc,
			true,
			false,
		},
		{
			"V1beta1 is not available",
			v1AvailableDisc,
			false,
			false,
		},
		{
			"error when fetching resources",
			v1beta1ErrOnServerResourcesDisc,
			false,
			true,
		},
	}

	for _, tc := range testCases {
		hasIngress, err := discovery.HasV1beta1Ingress(tc.client)
		s.Equal(tc.want, hasIngress)
		s.Equal(tc.wantErr, err != nil)
	}
}

func (s *DiscoveryTestSuite) TestHasV1Ingress() {
	testCases := []struct {
		name    string
		client  k8sdisc.DiscoveryInterface
		want    bool
		wantErr bool
	}{
		{
			"V1 is available",
			v1AvailableDisc,
			true,
			false,
		},
		{
			"V1 is not available",
			v1beta1AvailableDisc,
			false,
			false,
		},
		{
			"error when fetching resources",
			v1ErrOnServerResourcesDisc,
			false,
			true,
		},
	}

	for _, tc := range testCases {
		hasIngress, err := discovery.HasV1Ingress(tc.client)
		s.Equal(tc.want, hasIngress)
		s.Equal(tc.wantErr, err != nil)
	}
}
