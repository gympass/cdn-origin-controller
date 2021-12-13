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

package controllers

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/Gympass/cdn-origin-controller/api/v1alpha1"
)

func TestRunIngressReconcilerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &IngressReconcilerSuite{})
}

type IngressReconcilerSuite struct {
	suite.Suite
}

func (s IngressReconcilerSuite) Test_ingressParamsForUserOrigins_Success() {
	testCases := []struct {
		name            string
		annotationValue string
		expectedParams  []ingressParams
	}{
		{
			name:            "Has no annotation",
			annotationValue: "",
			expectedParams:  nil,
		},
		{
			name: "Has a single user origin",
			annotationValue: `
                                - host: foo.com
                                  responseTimeout: 35
                                  paths:
                                    - /foo
                                    - /foo/*
                                  viewerFunctionARN: foo
                                  originRequestPolicy: None`,
			expectedParams: []ingressParams{
				{
					group:             "group",
					destinationHost:   "foo.com",
					paths:             []path{{pathPattern: "/foo"}, {pathPattern: "/foo/*"}},
					originRespTimeout: int64(35),
					viewerFnARN:       "foo",
					originReqPolicy:   "None",
				},
			},
		},
		{
			name: "Has multiple user origins",
			annotationValue: `
                                - host: foo.com
                                  paths:
                                    - /foo
                                  viewerFunctionARN: foo
                                  originRequestPolicy: None
                                - host: bar.com
                                  responseTimeout: 35
                                  paths:
                                    - /bar`,
			expectedParams: []ingressParams{
				{
					group:           "group",
					destinationHost: "foo.com",
					paths:           []path{{pathPattern: "/foo"}},
					originReqPolicy: "None",
					viewerFnARN:     "foo",
				},
				{
					group:             "group",
					destinationHost:   "bar.com",
					paths:             []path{{pathPattern: "/bar"}},
					originRespTimeout: int64(35),
				},
			},
		},
	}

	r := &IngressReconciler{log: logr.DiscardLogger{}}
	for _, tc := range testCases {
		ing := &networkingv1.Ingress{}
		if len(tc.annotationValue) > 0 {
			ing.Annotations = map[string]string{cfUserOriginsAnnotation: tc.annotationValue}
		}

		got, err := r.ingressParamsForUserOrigins("group", ing)
		s.NoError(err, "test: %s", tc.name)
		s.Equal(tc.expectedParams, got, "test: %s", tc.name)
	}
}

func (s IngressReconcilerSuite) Test_ingressParamsForUserOrigins_InvalidAnnotationValue() {
	testCases := []struct {
		name            string
		annotationValue string
	}{
		{
			name:            "No path",
			annotationValue: "foo.com:",
		},
		{
			name:            "No host",
			annotationValue: ":/bar",
		},
		{
			name:            "':' used instead of comma",
			annotationValue: "foo.com:/bar:another-foo.com:/another-bar",
		},
	}

	r := &IngressReconciler{log: logr.DiscardLogger{}}
	for _, tc := range testCases {
		ing := &networkingv1.Ingress{}
		ing.Annotations = map[string]string{cfUserOriginsAnnotation: tc.annotationValue}

		got, err := r.ingressParamsForUserOrigins("group", ing)
		s.Error(err, "test: %s", tc.name)
		s.Nil(got, "test: %s", tc.name)
	}
}

func (s IngressReconcilerSuite) Test_getDeletions() {
	testCases := []struct {
		name             string
		desired, current []string
		want             []string
	}{
		{
			name:    "No current state",
			desired: []string{"foo"},
			current: nil,
			want:    nil,
		},
		{
			name:    "Current and desired state match",
			desired: []string{"foo"},
			current: []string{"foo"},
			want:    nil,
		},
		{
			name:    "Current and desired state don't match",
			desired: []string{"foo"},
			current: []string{"bar"},
			want:    []string{"bar"},
		},
		{
			name:    "No desired state and has current state",
			desired: nil,
			current: []string{"foo"},
			want:    []string{"foo"},
		},
	}

	for _, tc := range testCases {
		s.Equal(tc.want, getDeletions(tc.desired, tc.current), "test: %s", tc.name)
	}
}

type nsName struct {
	namespace, name string
}

func (s IngressReconcilerSuite) Test_filterIngressRef() {
	testCases := []struct {
		name     string
		toFilter nsName
		data     []nsName
		want     []nsName
	}{
		{
			name:     "Empty refs",
			toFilter: nsName{"to", "filter"},
			data:     []nsName{},
			want:     nil,
		},
		{
			name:     "Non-empty refs, nothing to filter",
			toFilter: nsName{"to", "filter"},
			data:     []nsName{{"baz", "foo"}},
			want:     []nsName{{"baz", "foo"}},
		},
		{
			name:     "Refs only have what should be filtered",
			toFilter: nsName{"to", "filter"},
			data:     []nsName{{"to", "filter"}},
			want:     nil,
		},
		{
			name:     "Refs have what should be filtered and more",
			toFilter: nsName{"to", "filter"},
			data: []nsName{
				{"to", "filter"},
				{"foo", "baz"},
			},
			want: []nsName{{"foo", "baz"}},
		},
	}

	for _, tc := range testCases {
		toFilter := &networkingv1.Ingress{}
		toFilter.Name = tc.toFilter.name
		toFilter.Namespace = tc.toFilter.namespace

		cdnStatus := newCDNStatus(tc.data)
		result := filterIngressRef(&cdnStatus, toFilter)

		want := newCDNStatus(tc.want)
		s.Equal(want, result, "test: %s", tc.name)
	}
}

func newCDNStatus(data []nsName) v1alpha1.CDNStatus {
	cdnStatus := &v1alpha1.CDNStatus{Status: v1alpha1.CDNStatusStatus{Ingresses: make(v1alpha1.IngressRefs)}}
	for _, it := range data {
		cdnStatus.Status.Ingresses[v1alpha1.NewIngressRef(it.namespace, it.name)] = "Synced"
	}
	return *cdnStatus
}
