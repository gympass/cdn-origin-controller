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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/acm/acmiface"
)

var (
	errFindCert = errors.New("finding certificate")
)

// CertFilter type represents a certificate filter interface
type CertFilter func(c Certificate) bool

// Repository provides methods for manipulating Custom domain names on AWS
type Repository interface {
	FindByFilter(CertFilter) ([]Certificate, error)
}

type acmCertRepository struct {
	client acmiface.ACMAPI
}

// NewRepository creates a new Repository
func NewRepository(c acmiface.ACMAPI) Repository {
	return acmCertRepository{client: c}
}

// FindByFilter find a certificate given a filter
func (r acmCertRepository) FindByFilter(filter CertFilter) ([]Certificate, error) {

	input := &acm.ListCertificatesInput{
		CertificateStatuses: aws.StringSlice([]string{acm.CertificateStatusIssued}),
	}

	var certs []Certificate
	var certDiscoveryErr error

	err := r.client.ListCertificatesPages(input, func(output *acm.ListCertificatesOutput, _ bool) bool {
		for _, acmCertSummary := range output.CertificateSummaryList {
			acmCert, err := r.client.DescribeCertificate(&acm.DescribeCertificateInput{
				CertificateArn: acmCertSummary.CertificateArn,
			})

			if err != nil {
				certDiscoveryErr = fmt.Errorf("describing certificate (ARN: %s): %v", *acmCertSummary.CertificateArn, err)
				return false
			}

			certDetails := acmCert.Certificate
			dnCert := New(*certDetails.CertificateArn,
				*certDetails.DomainName,
				aws.StringValueSlice(acmCert.Certificate.SubjectAlternativeNames),
			)
			if filter(dnCert) {
				certs = append(certs, dnCert)
			}
		}
		return true
	})

	if certDiscoveryErr != nil {
		err = certDiscoveryErr
	}
	if err != nil {
		return []Certificate{}, fmt.Errorf("%w: %v", errFindCert, err)
	}

	return certs, nil
}
