package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRunCDNStatusTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &CDNStatusTestSuite{})
}

type CDNStatusTestSuite struct {
	suite.Suite
}

func (s *CDNStatusTestSuite) Test_SetIngressRef_IngressAlreadyExists() {
	key := IngressRef("namespace/name")
	cdnStatus := CDNStatus{
		Status: CDNStatusStatus{
			Ingresses: IngressRefs{
				key: failedIngressStatus,
			},
		},
	}

	nsName := &metav1.ObjectMeta{Namespace: "namespace", Name: "name"}
	cdnStatus.SetIngressRef(true, nsName)

	s.Len(cdnStatus.Status.Ingresses, 1)
	s.Equal(syncedIngressStatus, cdnStatus.Status.Ingresses[key])
}

func (s *CDNStatusTestSuite) Test_SetIngressRef_AddFirstIngress() {
	cdnStatus := CDNStatus{}

	nsName := &metav1.ObjectMeta{Namespace: "bar", Name: "foo"}
	cdnStatus.SetIngressRef(true, nsName)

	key := IngressRef("bar/foo")

	s.Len(cdnStatus.Status.Ingresses, 1)
	s.Equal(syncedIngressStatus, cdnStatus.Status.Ingresses[key])
}

func (s *CDNStatusTestSuite) Test_SetIngressRef_AddNewIngressToExistingIngresses() {
	existingKey := IngressRef("namespace/name")
	cdnStatus := CDNStatus{
		Status: CDNStatusStatus{
			Ingresses: IngressRefs{
				existingKey: failedIngressStatus,
			},
		},
	}

	nsName := &metav1.ObjectMeta{Namespace: "bar", Name: "foo"}
	cdnStatus.SetIngressRef(true, nsName)

	newKey := IngressRef("bar/foo")

	s.Len(cdnStatus.Status.Ingresses, 2)
	s.Equal(failedIngressStatus, cdnStatus.Status.Ingresses[existingKey])
	s.Equal(syncedIngressStatus, cdnStatus.Status.Ingresses[newKey])
}

func (s *CDNStatusTestSuite) Test_GetIngressKeys() {
	testCases := []struct {
		name      string
		cdnStatus CDNStatus
		want      []client.ObjectKey
	}{
		{
			name:      "No Ingresses",
			cdnStatus: CDNStatus{},
			want:      nil,
		},
		{
			name: "Two Ingresses",
			cdnStatus: CDNStatus{
				Status: CDNStatusStatus{
					Ingresses: IngressRefs{
						"bar/foo": failedIngressStatus,
						"foo/bar": syncedIngressStatus,
					},
				},
			},
			want: []types.NamespacedName{
				{
					Namespace: "bar",
					Name:      "foo",
				},
				{
					Namespace: "foo",
					Name:      "bar",
				},
			},
		},
	}

	for _, tc := range testCases {
		got := tc.cdnStatus.GetIngressKeys()
		s.ElementsMatchf(tc.want, got, "test case: %s", tc.name)
	}
}
