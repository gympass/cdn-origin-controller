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
	"strings"

	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/spf13/viper"
)

const (
	logLevelKey              = "log_level"
	devModeKey               = "dev_mode"
	cfDefaultOriginDomain    = "cf_default_origin_domain"
	cfPriceClassKey          = "cf_price_class"
	cfWafArnKey              = "cf_aws_waf"
	cfCustomSSLCertArnKey    = "cf_custom_ssl_cert"
	cfSecurityPolicyKey      = "cf_security_policy"
	cfEnableLoggingKey       = "cf_enable_logging"
	cfS3BucketLogKey         = "cf_s3_bucket_log"
	cfEnableIPV6Key          = "cf_enable_ipv6"
	cfDescriptionTemplateKey = "cf_description_template"
	cfAliasCreationKey       = "cf_alias_creation"
	cfCustomTagsKey          = "cf_custom_tags"
)

func init() {
	viper.SetDefault(logLevelKey, "info")
	viper.SetDefault(devModeKey, "false")
	viper.SetDefault(cfDefaultOriginDomain, "")
	viper.SetDefault(cfPriceClassKey, awscloudfront.PriceClassPriceClassAll)
	viper.SetDefault(cfWafArnKey, "")
	viper.SetDefault(cfCustomSSLCertArnKey, "")
	viper.SetDefault(cfSecurityPolicyKey, "")
	viper.SetDefault(cfEnableLoggingKey, "false")
	viper.SetDefault(cfS3BucketLogKey, "")
	viper.SetDefault(cfEnableIPV6Key, "true")
	viper.SetDefault(cfDescriptionTemplateKey, "Serve contents for {{group}} group.")
	viper.SetDefault(cfAliasCreationKey, "true")
	viper.SetDefault(cfCustomTagsKey, "")
	viper.AutomaticEnv()
}

// Config represents all possible configurations for the Operator
type Config struct {
	// LogLevel represents log verbosity. Overridden to "debug" if DevMode is true.
	LogLevel string
	// DevMode when set to "true" logs in unstructured text instead of JSON.
	DevMode bool
	// DefaultOriginDomain represents a valid domain to define in default origin.
	DefaultOriginDomain string
	// CloudFrontPriceClass determines how many edge locations CloudFront will use for your distribution.
	// ref: https://docs.aws.amazon.com/sdk-for-go/api/service/cloudfront/
	CloudFrontPriceClass string
	// CloudFrontWAFARN the Web ACL ARN.
	CloudFrontWAFARN string
	// CloudFrontCustomSSLCertARN the ACM certificate ARN.
	CloudFrontCustomSSLCertARN string
	// CloudFrontSecurityPolicy the minimum SSL/TLS protocol that CloudFront can use to communicate with viewers.
	// ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-viewercertificate.html
	CloudFrontSecurityPolicy string
	// CloudFrontEnableLogging if should enable cloudfront logging.
	CloudFrontEnableLogging bool
	// CloudFrontS3BucketLog if logging enabled represents the S3 Bucket URL to persists, for example myawslogbucket.s3.amazonaws.com.
	CloudFrontS3BucketLog string
	// CloudFrontEnableIPV6 if should enable ipv6 for distribution responses.
	CloudFrontEnableIPV6 bool
	// CloudFrontDescriptionTemplate the description template for distribution.
	CloudFrontDescriptionTemplate string
	// CloudFrontAliasCreation if should create a DNS alias for distribution.
	CloudFrontAliasCreation bool
	// CloudFrontCustomTags all custom tags that will be persisted to distribution.
	CloudFrontCustomTags map[string]string
}

// Parse environment variables into a config struct
func Parse() Config {
	devMode := viper.GetBool(devModeKey)
	logLvl := viper.GetString(logLevelKey)
	if devMode {
		logLvl = "debug"
	}

	return Config{
		LogLevel:                      logLvl,
		DevMode:                       devMode,
		DefaultOriginDomain:           viper.GetString(cfDefaultOriginDomain),
		CloudFrontPriceClass:          viper.GetString(cfPriceClassKey),
		CloudFrontWAFARN:              viper.GetString(cfWafArnKey),
		CloudFrontCustomSSLCertARN:    viper.GetString(cfCustomSSLCertArnKey),
		CloudFrontSecurityPolicy:      viper.GetString(cfSecurityPolicyKey),
		CloudFrontEnableLogging:       viper.GetBool(cfEnableLoggingKey),
		CloudFrontS3BucketLog:         viper.GetString(cfS3BucketLogKey),
		CloudFrontEnableIPV6:          viper.GetBool(cfEnableIPV6Key),
		CloudFrontDescriptionTemplate: viper.GetString(cfDescriptionTemplateKey),
		CloudFrontAliasCreation:       viper.GetBool(cfAliasCreationKey),
		CloudFrontCustomTags:          extractTags(viper.GetString(cfCustomTagsKey)),
	}
}

func extractTags(customTags string) map[string]string {
	m := make(map[string]string)
	if len(customTags) == 0 {
		return m
	}

	tags := strings.Split(customTags, ",")
	for _, pair := range tags {
		tag := strings.Split(pair, "=")
		m[tag[0]] = tag[1]
	}
	return m
}
