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

package route53

import (
	"strings"

	awsroute53 "github.com/aws/aws-sdk-go/service/route53"
)

// Entry represents an alias entry with all desired record types for it
type Entry struct {
	Name string
	Type []string
}

// Aliases represents all aliases which should be bound to a CF distribution
type Aliases struct {
	Target  string
	Entries []Entry
}

// Domains returns a slice of all domains from an Aliases' Entries
func (a Aliases) Domains() []string {
	var domains []string

	for _, e := range a.Entries {
		domains = append(domains, e.Name)
	}
	return domains
}

// NewAliases builds a new Aliases
func NewAliases(target string, domains []string, ipv6Enabled bool) Aliases {
	aliases := Aliases{
		Target: target,
	}

	types := []string{awsroute53.RRTypeA}
	if ipv6Enabled {
		types = append(types, awsroute53.RRTypeAaaa)
	}

	for _, domain := range domains {
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}

		entry := Entry{
			Name: domain,
			Type: types,
		}

		aliases.Entries = append(aliases.Entries, entry)
	}

	return aliases
}
