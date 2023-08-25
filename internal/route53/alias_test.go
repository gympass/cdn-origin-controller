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

package route53_test

import (
	"testing"

	awsroute53 "github.com/aws/aws-sdk-go/service/route53"
	"github.com/stretchr/testify/suite"

	"github.com/Gympass/cdn-origin-controller/internal/route53"
)

func TestRunAliasesTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &AliasesTestSuite{})
}

type AliasesTestSuite struct {
	suite.Suite
}

func (s *AliasesTestSuite) TestNewAliases_IPV6Disabled() {
	ipv6Enabled := false
	target := "target.foo.bar."
	domains := []string{"alias.foo.bar."}
	zoneID := "A3GK5SAMPLE"

	aliases := route53.NewAliases(target, zoneID, domains, ipv6Enabled)
	s.Equal(target, aliases.Target)
	s.Len(aliases.Entries, 1)
	s.Equal(domains[0], aliases.Entries[0].Name)
	s.ElementsMatch(aliases.Entries[0].Types, []string{awsroute53.RRTypeA})
}

func (s *AliasesTestSuite) TestNewAliases_IPV6Enabled() {
	ipv6Enabled := true
	target := "target.foo.bar."
	domains := []string{"alias.foo.bar."}
	zoneID := "A3GK5SAMPLE"

	aliases := route53.NewAliases(target, zoneID, domains, ipv6Enabled)
	s.Equal(target, aliases.Target)
	s.Len(aliases.Entries, 1)
	s.Equal(domains[0], aliases.Entries[0].Name)
	s.ElementsMatch(aliases.Entries[0].Types, []string{awsroute53.RRTypeA, awsroute53.RRTypeAaaa})
}

func (s *AliasesTestSuite) TestNewAliases_MultipleAliases() {
	ipv6Enabled := false
	target := "target.foo.bar."
	domains := []string{"alias1.foo.bar.", "alias2.foo.bar."}
	zoneID := "A3GK5SAMPLE"

	aliases := route53.NewAliases(target, zoneID, domains, ipv6Enabled)
	s.Equal(target, aliases.Target)

	expectedEntries := []route53.Entry{
		{
			Name:  "alias1.foo.bar.",
			Types: []string{awsroute53.RRTypeA},
		},
		{
			Name:  "alias2.foo.bar.",
			Types: []string{awsroute53.RRTypeA},
		},
	}
	s.ElementsMatch(expectedEntries, aliases.Entries)
}

func (s *AliasesTestSuite) TestNewAliases_DenormalizedInputDomains() {
	ipv6Enabled := false
	target := "target.foo.bar."
	domains := []string{"alias1.foo.bar", "alias2.foo.bar"}
	zoneID := "A3GK5SAMPLE"

	aliases := route53.NewAliases(target, zoneID, domains, ipv6Enabled)
	s.Equal(target, aliases.Target)

	expectedEntries := []route53.Entry{
		{
			Name:  "alias1.foo.bar.",
			Types: []string{awsroute53.RRTypeA},
		},
		{
			Name:  "alias2.foo.bar.",
			Types: []string{awsroute53.RRTypeA},
		},
	}
	s.ElementsMatch(expectedEntries, aliases.Entries)
}

func (s *AliasesTestSuite) TestDomains() {
	testCases := []struct {
		name            string
		expectedDomains []string
	}{
		{
			"No domains configured",
			nil,
		},
		{
			"Multiple domains configured",
			[]string{"alias1.foo.bar.", "alias2.foo.bar."},
		},
	}

	for _, tc := range testCases {
		aliases := route53.NewAliases("target.foo.bar.", "A3GK5SAMPLE", tc.expectedDomains, false)
		s.ElementsMatch(tc.expectedDomains, aliases.Domains(), "test: %s", tc.name)
	}
}

func TestRunAliasTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &AliasTestSuite{})
}

type AliasTestSuite struct {
	suite.Suite
}

func (s *AliasTestSuite) TestNormalizeDomains() {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "Inputs have a '.' at the end",
			input:    []string{"alias1.foo.bar.", "alias2.foo.bar."},
			expected: []string{"alias1.foo.bar.", "alias2.foo.bar."},
		},
		{
			name:     "Part of inputs don't have a '.' at the end",
			input:    []string{"alias1.foo.bar.", "alias2.foo.bar"},
			expected: []string{"alias1.foo.bar.", "alias2.foo.bar."},
		},
		{
			name:     "No inputs have a '.' at the end",
			input:    []string{"alias1.foo.bar", "alias2.foo.bar"},
			expected: []string{"alias1.foo.bar.", "alias2.foo.bar."},
		},
	}

	for _, tc := range testCases {
		normalized := route53.NormalizeDomains(tc.input)
		s.ElementsMatch(normalized, tc.expected, "test: %s", tc.name)
	}
}
