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
	"fmt"
	"strings"

	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/spf13/viper"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	logLevelKey                                   = "log_level"
	devModeKey                                    = "dev_mode"
	enableDeletionKey                             = "enable_deletion"
	cfDefaultOriginDomainKey                      = "cf_default_origin_domain"
	cfPriceClassKey                               = "cf_price_class"
	cfWafArnKey                                   = "cf_aws_waf"
	cfSecurityPolicyKey                           = "cf_security_policy"
	cfEnableLoggingKey                            = "cf_enable_logging"
	cfS3BucketLogKey                              = "cf_s3_bucket_log"
	cfS3BucketLogPrefixKey                        = "cf_s3_bucket_log_prefix"
	cfEnableIPV6Key                               = "cf_enable_ipv6"
	cfDescriptionTemplateKey                      = "cf_description_template"
	cfCustomTagsKey                               = "cf_custom_tags"
	cfDefaultCachingPolicyIDKey                   = "cf_default_caching_policy_id"
	cfDefaultCacheRequestPolicyIDKey              = "cf_default_cache_request_policy_id"
	cfDefaultPublicOriginAccessRequestPolicyIDKey = "cf_default_public_origin_access_request_policy_id"
	cfDefaultBucketOriginAccessRequestPolicyIDKey = "cf_default_bucket_origin_access_request_policy_id"
	createBlockedKey                              = "block_creation"
	createBlockedAllowListKey                     = "block_creation_allow_list"
)

func init() {
	initDefaults()
}

func initDefaults() {
	viper.SetDefault(logLevelKey, "info")
	viper.SetDefault(devModeKey, "false")
	viper.SetDefault(enableDeletionKey, "false")
	viper.SetDefault(cfDefaultOriginDomainKey, "")
	viper.SetDefault(cfPriceClassKey, awscloudfront.PriceClassPriceClassAll)
	viper.SetDefault(cfWafArnKey, "")
	viper.SetDefault(cfSecurityPolicyKey, "")
	viper.SetDefault(cfEnableLoggingKey, "false")
	viper.SetDefault(cfS3BucketLogKey, "")
	viper.SetDefault(cfS3BucketLogPrefixKey, "")
	viper.SetDefault(cfEnableIPV6Key, "true")
	viper.SetDefault(cfDescriptionTemplateKey, "Serve contents for {{group}} group.")
	viper.SetDefault(cfCustomTagsKey, "")
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-cache-policies.html
	// Default is caching disabled
	viper.SetDefault(cfDefaultCachingPolicyIDKey, "4135ea2d-6df8-44a3-9df3-4b5a84be39ad")
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-origin-request-policies.html#managed-origin-request-policy-all-viewer
	// Default is all viewer
	viper.SetDefault(cfDefaultCacheRequestPolicyIDKey, "216adef6-5c7f-47e4-b989-5492eafa07d3")
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-origin-request-policies.html#managed-origin-request-policy-all-viewer
	// Default is all viewer
	viper.SetDefault(cfDefaultPublicOriginAccessRequestPolicyIDKey, "216adef6-5c7f-47e4-b989-5492eafa07d3")
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-origin-request-policies.html#managed-origin-request-policy-cors-s3
	// Default is CORS S3
	viper.SetDefault(cfDefaultBucketOriginAccessRequestPolicyIDKey, "88a5eaf4-2fd4-4709-b370-b4c650ea3fcf")
	viper.SetDefault(createBlockedKey, false)

	viper.AutomaticEnv()
}

// Config represents all possible configurations for the Operator
type Config struct {
	// LogLevel represents log verbosity. Overridden to "debug" if DevMode is true.
	LogLevel string
	// DevMode when set to "true" logs in unstructured text instead of JSON.
	DevMode bool
	// DeletionEnabled represent whether external components should be deleted based on K8s resources deletion
	DeletionEnabled bool
	// DefaultOriginDomain represents a valid domain to define in default origin.
	DefaultOriginDomain string
	// CloudFrontPriceClass determines how many edge locations CloudFront will use for your distribution.
	// ref: https://docs.aws.amazon.com/sdk-for-go/api/service/cloudfront/
	CloudFrontPriceClass string
	// CloudFrontWAFARN the Web ACL ARN.
	CloudFrontWAFARN string
	// CloudFrontSecurityPolicy the minimum SSL/TLS protocol that CloudFront can use to communicate with viewers.
	// ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-viewercertificate.html
	CloudFrontSecurityPolicy string
	// CloudFrontEnableLogging if should enable cloudfront logging.
	CloudFrontEnableLogging bool
	// CloudFrontS3BucketLog if logging enabled represents the S3 Bucket URL to persists, for example myawslogbucket.s3.amazonaws.com.
	CloudFrontS3BucketLog string
	// CloudFrontS3BucketLogPrefix is the prefix that should be added to the S3 path when sending logs. The directory on which logs should be stored.
	// Trailing slashes are ignored ("foo/bar/" is the same as "foo/bar").
	CloudFrontS3BucketLogPrefix string
	// CloudFrontEnableIPV6 if should enable ipv6 for distribution responses.
	CloudFrontEnableIPV6 bool
	// CloudFrontDescriptionTemplate the description template for distribution.
	CloudFrontDescriptionTemplate string
	// CloudFrontCustomTags all custom tags that will be persisted to distribution.
	CloudFrontCustomTags map[string]string
	// CloudFrontDefaultCachingPolicyID is the default caching policy ID.
	CloudFrontDefaultCachingPolicyID string
	// CloudFrontDefaultDistributionRequestPolicyID is the default request policy for distributions.
	CloudFrontDefaultCacheRequestPolicyID string
	// CloudFrontDefaultPublicOriginAccessRequestPolicyID is the default request policy for public origin access.
	CloudFrontDefaultPublicOriginAccessRequestPolicyID string
	// CloudFrontDefaultBucketOriginAccessRequestPolicyID is the default request policy for bucket origin access.
	CloudFrontDefaultBucketOriginAccessRequestPolicyID string
	// IsCreateBlocked configure whether to block creation of new CloudFront distributions. Useful when phasing out clusters or accounts, for example
	IsCreateBlocked bool
	// CreateAllowList holds a list of Ingresses namespaced names for which we should allow creation, even if IsCreateBlocked is true
	CreateAllowList []types.NamespacedName
}

// TLSIsEnabled returns whether TLS is enabled
func (c Config) TLSIsEnabled() bool {
	return len(c.CloudFrontSecurityPolicy) > 0
}

// IsCreationAllowed returns whether the creation of a new CloudFront distribution for the given Ingress should be allowed
func (c Config) IsCreationAllowed(ing *networkingv1.Ingress) bool {
	if !c.IsCreateBlocked {
		return true
	}

	ingName := types.NamespacedName{
		Namespace: ing.Namespace,
		Name:      ing.Name,
	}

	for _, candidate := range c.CreateAllowList {
		if candidate == ingName {
			return true
		}
	}

	return false
}

// Parse environment variables into a config struct
func Parse() (Config, error) {
	devMode := viper.GetBool(devModeKey)
	logLvl := viper.GetString(logLevelKey)
	if devMode {
		logLvl = "debug"
	}

	createAllowList, err := parseNamespacedNames(parseList(viper.GetString(createBlockedAllowListKey)))
	if err != nil {
		return Config{}, fmt.Errorf("invalid %q: %v", createBlockedAllowListKey, err)
	}

	return Config{
		LogLevel:                              logLvl,
		DevMode:                               devMode,
		DefaultOriginDomain:                   viper.GetString(cfDefaultOriginDomainKey),
		DeletionEnabled:                       viper.GetBool(enableDeletionKey),
		CloudFrontPriceClass:                  viper.GetString(cfPriceClassKey),
		CloudFrontWAFARN:                      viper.GetString(cfWafArnKey),
		CloudFrontSecurityPolicy:              viper.GetString(cfSecurityPolicyKey),
		CloudFrontEnableLogging:               viper.GetBool(cfEnableLoggingKey),
		CloudFrontS3BucketLog:                 viper.GetString(cfS3BucketLogKey),
		CloudFrontS3BucketLogPrefix:           removeTrailingSlash(viper.GetString(cfS3BucketLogPrefixKey)),
		CloudFrontEnableIPV6:                  viper.GetBool(cfEnableIPV6Key),
		CloudFrontDescriptionTemplate:         viper.GetString(cfDescriptionTemplateKey),
		CloudFrontCustomTags:                  extractTags(viper.GetString(cfCustomTagsKey)),
		CloudFrontDefaultCachingPolicyID:      viper.GetString(cfDefaultCachingPolicyIDKey),
		CloudFrontDefaultCacheRequestPolicyID: viper.GetString(cfDefaultCacheRequestPolicyIDKey),
		IsCreateBlocked:                       viper.GetBool(createBlockedKey),
		CreateAllowList:                       createAllowList,
		CloudFrontDefaultPublicOriginAccessRequestPolicyID: viper.GetString(cfDefaultPublicOriginAccessRequestPolicyIDKey),
		CloudFrontDefaultBucketOriginAccessRequestPolicyID: viper.GetString(cfDefaultBucketOriginAccessRequestPolicyIDKey),
	}, nil
}

func removeTrailingSlash(getString string) string {
	if strings.HasSuffix(getString, "/") {
		return getString[:len(getString)-1]
	}
	return getString
}

func parseNamespacedNames(names []string) ([]types.NamespacedName, error) {
	var result []types.NamespacedName
	for _, n := range names {
		name, err := nsName(n)
		if err != nil {
			return nil, fmt.Errorf("parsing namespaced name: %v", err)
		}
		result = append(result, name)
	}
	return result, nil
}

func nsName(name string) (types.NamespacedName, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return types.NamespacedName{}, fmt.Errorf(`namespaced name is not in "namespace/name" format: %q`, name)
	}
	return types.NamespacedName{
		Namespace: parts[0],
		Name:      parts[1],
	}, nil
}

func parseList(l string) []string {
	if l == "" {
		return nil
	}
	return strings.Split(l, ",")
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
