package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunCDNStatusTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &CDNStatusTestSuite{})
}

type CDNStatusTestSuite struct {
	suite.Suite
}

func (s *CDNStatusTestSuite) Test_SetRef_IngressAlreadyExists() {
	key := "name/namespace"
	cdnStatus := CDNStatus{
		Status: CDNStatusStatus{
			Ingresses: IngressRefs{
				key: "failed",
			},
		},
	}

	nsName := &metav1.ObjectMeta{Name: "name", Namespace: "namespace"}
	cdnStatus.SetRef(true, nsName)

	value, ok := cdnStatus.Status.Ingresses[key]
	s.T().Logf("%s %v", value, ok)

	s.Len(cdnStatus.Status.Ingresses, 1)
	s.Equal("synced", cdnStatus.Status.Ingresses[key])
}

func (s *CDNStatusTestSuite) Test_SetRef_AddFirstIngress() {
	cdnStatus := CDNStatus{}

	nsName := &metav1.ObjectMeta{Name: "foo", Namespace: "bar"}
	cdnStatus.SetRef(true, nsName)

	key := "foo/bar"

	s.Len(cdnStatus.Status.Ingresses, 1)
	s.Equal("synced", cdnStatus.Status.Ingresses[key])
}

func (s *CDNStatusTestSuite) Test_SetRef_AddNewIngressToExistingIngresses() {
	existingKey := "name/namespace"
	cdnStatus := CDNStatus{
		Status: CDNStatusStatus{
			Ingresses: IngressRefs{
				existingKey: "failed",
			},
		},
	}

	nsName := &metav1.ObjectMeta{Name: "foo", Namespace: "bar"}
	cdnStatus.SetRef(true, nsName)

	newKey := "foo/bar"

	s.Len(cdnStatus.Status.Ingresses, 2)
	s.Equal("failed", cdnStatus.Status.Ingresses[existingKey])
	s.Equal("synced", cdnStatus.Status.Ingresses[newKey])
}
