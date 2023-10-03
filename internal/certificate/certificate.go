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

// New creates a Certificate
func New(arn, domainName string, alternativeNames []string /*, renewalEligibility string*/) Certificate {
	return Certificate{
		arn:              arn,
		domainName:       domainName,
		alternativeNames: alternativeNames,
	}
}

// Certificate represents a basic certificate
type Certificate struct {
	arn              string
	domainName       string
	alternativeNames []string
}

// DomainName returns the main certificate domain name
func (c Certificate) DomainName() string {
	return c.domainName
}

// AlternativeNames returns a list of certificate subject alternative names
func (c Certificate) AlternativeNames() []string {
	return c.alternativeNames
}

// ARN returns the certificate identifier
func (c Certificate) ARN() string {
	return c.arn
}
