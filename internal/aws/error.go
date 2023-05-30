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
