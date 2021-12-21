package strhelper

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStringHelpersTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &StringHelpersTestSuite{})
}

type StringHelpersTestSuite struct {
	suite.Suite
}

func (s *StringHelpersTestSuite) TestSet_Add() {
	ss := NewSet()
	ss.Add("test-1")
	ss.Add("test-2")

	s.True(ss["test-1"])
	s.True(ss["test-2"])
}

func (s *StringHelpersTestSuite) TestSet_Contains() {
	ss := NewSet()
	ss.Add("test-1")
	ss.Add("test-2")

	s.True(ss.Contains("test-1"))
	s.True(ss.Contains("test-2"))
	s.False(ss.Contains("some other string"))
}

func (s *StringHelpersTestSuite) TestSet_ToSlice() {
	ss := NewSet()
	ss.Add("test-1")
	ss.Add("test-2")

	s.ElementsMatch([]string{"test-1", "test-2"}, ss.ToSlice())
}
