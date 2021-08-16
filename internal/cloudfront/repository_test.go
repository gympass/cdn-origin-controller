package cloudfront_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestRunRepositoryTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &RepositoryTestSuite{})
}

type RepositoryTestSuite struct {
	suite.Suite
}

func (s *RepositoryTestSuite) Test_Test() {
	s.True(true)
}
