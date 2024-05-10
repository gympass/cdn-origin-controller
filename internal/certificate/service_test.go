// Copyright (c) 2023 GPBR Participacoes LTDA.
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

package certificate

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestRunCertificateServiceSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &CertificateServiceTestSuite{})
}

type CertificateServiceTestSuite struct {
	suite.Suite
}

func (s *CertificateServiceTestSuite) TestMatchDomainFilters_Matches() {
	testCases := []struct {
		name                   string
		certDomainName         string
		certAlternativeDomains []string
		distrDomains           []string
	}{
		{
			name:                   "Matching distribution domains using all certificate alternative domains",
			certDomainName:         "foo.com",
			certAlternativeDomains: []string{"foo.com", "*.foo.com"},
			distrDomains:           []string{"www.foo.com", "foo.com"},
		},
		{
			name:                   "Matching distribution domains using all certificate alternative domains (alternative order on distr domains)",
			certDomainName:         "foo.com",
			certAlternativeDomains: []string{"foo.com", "*.foo.com"},
			distrDomains:           []string{"foo.com", "www.foo.com"},
		},
		{
			name:                   "Matching distribution domains using all certificate alternative domains (alternative order on cert domains)",
			certDomainName:         "foo.com",
			certAlternativeDomains: []string{"*.foo.com", "foo.com"},
			distrDomains:           []string{"foo.com", "www.foo.com"},
		},
		{
			name:                   "Matching distribution domains when certificate has additional alternative domains",
			certDomainName:         "bar.com",
			certAlternativeDomains: []string{"*.foo.com", "bar.com", "*.baz.com"},
			distrDomains:           []string{"www.foo.com", "bar.com"},
		},
		{
			name:                   "Matching distribution domains exactly with certificates domains",
			certDomainName:         "bar.com",
			certAlternativeDomains: []string{"baz.com"},
			distrDomains:           []string{"bar.com", "baz.com"},
		},
	}

	for _, tc := range testCases {
		cert := New("arn:foo", tc.certDomainName, tc.certAlternativeDomains)
		filter := matchingDomainFilter(tc.distrDomains)
		s.Truef(filter(cert), "testCase: %s", tc.name)
	}
}

func (s *CertificateServiceTestSuite) TestMatchDomainFilters_DoesntMatch() {
	testCases := []struct {
		name                   string
		certDomainName         string
		certAlternativeDomains []string
		distrDomains           []string
	}{
		{
			name:                   "Doesn't Match anything",
			certDomainName:         "bar.com",
			certAlternativeDomains: []string{"bar.com", "*.bar.com"},
			distrDomains:           []string{"www.foo.com", "foo.com"},
		},
		{
			name:                   "Doesn't Match one domain",
			certDomainName:         "*.xpto.com",
			certAlternativeDomains: []string{"*.xpto.com"},
			distrDomains:           []string{"www.xpto.com", "xpto.com"},
		},
	}

	for _, tc := range testCases {
		cert := New("arn:foo", tc.certDomainName, tc.certAlternativeDomains)
		filter := matchingDomainFilter(tc.distrDomains)
		s.Falsef(filter(cert), "testCase: %s", tc.name)
	}
}
