// Copyright (c) 2021 GPBR Participacoes LTDA.
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

package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunConfigTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConfigTestSuite{})
}

type ConfigTestSuite struct {
	suite.Suite
}

func (s *ConfigTestSuite) SetupTest() {
	viper.Reset()
	initDefaults()
}

func (s *ConfigTestSuite) TestConfigWithCustomTagsParsed() {
	expected := map[string]string{
		"foo":  "bar",
		"area": "platform",
	}

	viper.Set("cf_custom_tags", "foo=bar,area=platform")

	cfg, err := Parse()

	s.Equal(expected, cfg.CloudFrontCustomTags)
	s.NoError(err)
}

func (s *ConfigTestSuite) TestConfigNoCustomTags() {
	expected := map[string]string{}

	viper.Set("cf_custom_tags", "")

	cfg, err := Parse()

	s.Equal(expected, cfg.CloudFrontCustomTags)
	s.NoError(err)
}

func (s *ConfigTestSuite) TestParse_DefaultToBlockCreationIsFalse() {
	cfg, err := Parse()

	s.NoError(err)
	s.False(cfg.IsCreateBlocked)
}

func (s *ConfigTestSuite) TestParse_BucketPrefixIsSet() {
	testCases := []struct {
		name   string
		prefix string
	}{
		{
			name:   "No trailing slash",
			prefix: "foo/bar",
		},
		{
			name:   "With trailing slash",
			prefix: "foo/bar/",
		},
	}

	for _, tc := range testCases {
		viper.Set("cf_s3_bucket_log_prefix", tc.prefix)

		cfg, err := Parse()

		s.NoErrorf(err, "test case: %s", tc.name)
		s.Equalf("foo/bar", cfg.CloudFrontS3BucketLogPrefix, "test case: %s", tc.name)
	}
}

func (s *ConfigTestSuite) TestIsCreationAllowed_UnblockedCreationReturnsTrue() {
	viper.Set(createBlockedKey, "false")

	cfg, err := Parse()
	s.NoError(err)

	s.True(cfg.IsCreationAllowed(&networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
		},
	}))
}

func (s *ConfigTestSuite) TestIsCreationAllowed_AllowedIngressWithBlockedCreationReturnsTrue() {
	viper.Set(createBlockedKey, "true")
	viper.Set(createBlockedAllowListKey, "ns/allowed")

	cfg, err := Parse()
	s.NoError(err)

	s.True(cfg.IsCreationAllowed(&networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "allowed",
		},
	}))
}

func (s *ConfigTestSuite) TestIsCreationAllowed_IngressNotOnAllowListWithBlockedCreationReturnsFalse() {
	viper.Set(createBlockedKey, "true")
	viper.Set(createBlockedAllowListKey, "ns/allowed")

	cfg, err := Parse()
	s.NoError(err)

	s.False(cfg.IsCreationAllowed(&networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "forbidden",
		},
	}))
}
