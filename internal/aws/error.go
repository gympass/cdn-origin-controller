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

package aws

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

// IsErrorCode returns whether the give error matches an AWS error with the give errCode.
// If the input error is not a valid awserr.Error, it returns false
func IsErrorCode(err error, errCode string) bool {
	var aerr awserr.Error
	if ok := errors.As(err, &aerr); !ok {
		return false
	}
	return aerr.Code() == errCode
}

// IgnoreErrorCode will return nil if the input error is a valid awserr.Error
// matching the given errCode. It will return the input error as-is otherwise
func IgnoreErrorCode(err error, errCode string) error {
	if IsErrorCode(err, errCode) {
		return nil
	}
	return err
}

// IgnoreErrorCodef will return nil if the input error is a valid awserr.Error
// matching the given errCode. It will return the input error formatted with the given
// format otherwise following fmt.Errorf behavior
func IgnoreErrorCodef(format string, err error, errCode string) error {
	if IsErrorCode(err, errCode) {
		return nil
	}
	return fmt.Errorf(format, err)
}
