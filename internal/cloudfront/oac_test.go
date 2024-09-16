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

package cloudfront

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestRunOACTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &OACTestSuite{})
}

type OACTestSuite struct {
	suite.Suite
}

func (s *OACTestSuite) TestNewOACWithDefaultNamePattern() {
	oac := NewOAC("asylium.gympass.com", "gympass-production-nv-wellz-accounts.s3.us-east-1.amazonaws.com")
	s.Equal("asylium.gympass.com-gympass-production-nv-wellz-accounts", oac.Name)
	s.True(len(oac.Name) <= 63)
}

func (s *OACTestSuite) TestNewOACWithShortNamePattern() {
	oac := NewOAC("wellz-accounts-fe-develop-nv.nv.dev.us.gympass.cloud", "gympass-develop-nv-wellz-accounts-fe-develop-nv.s3.us-east-1.amazonaws.com")
	s.True(strings.HasPrefix(oac.Name, "wellz-accounts-fe-develop-nv-"))
	s.True(len(oac.Name) <= 63)
}
