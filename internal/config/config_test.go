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

package config_test

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"

	"github.com/Gympass/cdn-origin-controller/internal/config"
)

func TestRunConfigTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConfigTestSuite{})
}

type ConfigTestSuite struct {
	suite.Suite
}

func (s *ConfigTestSuite) TestConfigWithCustomTagsParsed() {
	expected := map[string]string{
		"foo":  "bar",
		"area": "platform",
	}

	viper.Set("cf_custom_tags", "foo=bar,area=platform")

	cfg := config.Parse()

	s.Equal(expected, cfg.CloudFrontCustomTags)
}

func (s *ConfigTestSuite) TestConfigNoCustomTags() {
	expected := map[string]string{}

	viper.Set("cf_custom_tags", "")

	cfg := config.Parse()

	s.Equal(expected, cfg.CloudFrontCustomTags)
}
