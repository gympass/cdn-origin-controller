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
	cdnStatus := CDNStatus{
		Status: CDNStatusStatus{
			Ingresses: IngressRefs{
				{
					"name",
					"namespace",
					false,
				},
			},
		},
	}

	nsName := &metav1.ObjectMeta{Name: "name", Namespace: "namespace"}
	cdnStatus.SetRef(true, nsName)

	s.Len(cdnStatus.Status.Ingresses, 1)
	s.True(cdnStatus.Status.Ingresses[0].InSync)
}

func (s *CDNStatusTestSuite) Test_SetRef_AddFirstIngress() {
	cdnStatus := CDNStatus{}

	nsName := &metav1.ObjectMeta{Name: "foo", Namespace: "bar"}
	cdnStatus.SetRef(true, nsName)

	expectedNewIng := IngressRef{
		Name:      nsName.Name,
		Namespace: nsName.Namespace,
		InSync:    true,
	}
	s.Len(cdnStatus.Status.Ingresses, 1)
	s.Equal(expectedNewIng, cdnStatus.Status.Ingresses[0])
}

func (s *CDNStatusTestSuite) Test_SetRef_AddNewIngressToExistingIngresses() {
	existingIngRef := IngressRef{
		Name:      "name",
		Namespace: "namespace",
	}
	cdnStatus := CDNStatus{
		Status: CDNStatusStatus{
			Ingresses: IngressRefs{existingIngRef},
		},
	}

	nsName := &metav1.ObjectMeta{Name: "foo", Namespace: "bar"}
	cdnStatus.SetRef(true, nsName)

	expectedNewIng := IngressRef{
		Name:      nsName.Name,
		Namespace: nsName.Namespace,
		InSync:    true,
	}
	s.Len(cdnStatus.Status.Ingresses, 2)
	s.Equal(existingIngRef, cdnStatus.Status.Ingresses[0])
	s.Equal(expectedNewIng, cdnStatus.Status.Ingresses[1])
}
