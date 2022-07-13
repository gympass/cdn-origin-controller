// Copyright (c) 2022 GPBR Participacoes LTDA.
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

package test

import (
	"context"

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockK8sClient is a mocked client.Client to be used during testing
type MockK8sClient struct {
	mock.Mock
	ExpectedStatusWriter client.StatusWriter
	ExpectedScheme       *runtime.Scheme
	ExpectedRESTMapper   meta.RESTMapper
}

func (m *MockK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	called := m.Called(ctx, key, obj)
	return called.Error(0)
}

func (m *MockK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	called := m.Called(ctx, list, opts)
	return called.Error(0)
}

func (m *MockK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	called := m.Called(ctx, obj, opts)
	return called.Error(0)
}

func (m *MockK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	called := m.Called(ctx, obj, opts)
	return called.Error(0)
}

func (m *MockK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	called := m.Called(ctx, obj, opts)
	return called.Error(0)
}

func (m *MockK8sClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	called := m.Called(ctx, obj, patch, opts)
	return called.Error(0)
}

func (m *MockK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	called := m.Called(ctx, obj, opts)
	return called.Error(0)
}

func (m *MockK8sClient) Status() client.StatusWriter {
	return m.ExpectedStatusWriter
}

func (m *MockK8sClient) Scheme() *runtime.Scheme {
	return m.ExpectedScheme
}

func (m *MockK8sClient) RESTMapper() meta.RESTMapper {
	return m.ExpectedRESTMapper
}
