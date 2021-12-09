package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/types"
)

func TestRunIngressRefTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &IngressRefTestSuite{})
}

type IngressRefTestSuite struct {
	suite.Suite
}

func (s IngressRefTestSuite) TestGetName() {
	ref := NewIngressRef("namespace", "name")
	s.Equal("name", ref.GetName())
}

func (s IngressRefTestSuite) TestGetNamespace() {
	ref := NewIngressRef("namespace", "name")
	s.Equal("namespace", ref.GetNamespace())
}

func (s IngressRefTestSuite) TestToNamespacedName() {
	expected := types.NamespacedName{
		Namespace: "namespace",
		Name:      "name",
	}

	ref := NewIngressRef("namespace", "name")
	s.Equal(expected, ref.ToNamespacedName())
}
