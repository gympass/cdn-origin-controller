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
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNoMatchingCert any matching certificate error
	ErrNoMatchingCert = errors.New("could not find any matching certificate")
)

// Service handle the certificate actions as discovery
type Service interface {
	DiscoverByHost([]string) (Certificate, error)
}

// NewService creates a new Certificate Service
func NewService(c Repository) Service {
	return acmCertService{repo: c}
}

type acmCertService struct {
	repo Repository
}

// DiscoverByHost tries to discover a certificate given hosts
func (a acmCertService) DiscoverByHost(hosts []string) (Certificate, error) {

	certs, err := a.repo.FindByFilter(matchingDomainFilter(hosts))

	if err != nil {
		return Certificate{}, fmt.Errorf("discovery certificate: %v", err)
	}

	if len(certs) == 0 {
		return Certificate{}, ErrNoMatchingCert
	}

	return certs[0], nil
}

func matchingDomainFilter(hosts []string) CertFilter {
	return func(c Certificate) bool {
		for _, host := range hosts {
			if !certMatches(host, c) {
				return false
			}
		}
		return true
	}
}

func certMatches(distHost string, c Certificate) bool {
	for _, certHost := range append(c.AlternativeNames(), c.DomainName()) {
		if distHost == certHost {
			return true
		}
		hs := strings.Split(distHost, ".")
		hostDomain := strings.Join(hs[1:], ".")

		if strings.HasPrefix(certHost, "*.") {
			certHost = strings.ReplaceAll(certHost, "*.", "")
		}

		if certHost == hostDomain {
			return true
		}
	}

	return false
}
