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

package cloudfront

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"k8s.io/apimachinery/pkg/util/wait"

	cdnaws "github.com/Gympass/cdn-origin-controller/internal/aws"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-cache-policies.html
	cachingDisabledPolicyID = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-origin-request-policies.html
	allViewerOriginRequestPolicyID = "216adef6-5c7f-47e4-b989-5492eafa07d3"
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-origin-request-policies.html#managed-origin-request-policy-all-viewer-except-host-header
	allViewerExceptHostHeaderOriginRequestPolicyID = "b689b0a8-53d0-40ab-baf2-68738e2966ac"
)

const (
	originSSLProtocolTLSv12 = "TLSv1.2"
	originSSLProtocolTLSv11 = "TLSv1.1"
	originSSLProtocolTLSv1  = "TLSv1"
	originSSLProtocolSSLv3  = "SSLv3"
)

// ErrDistNotFound represents failure when finding/fetching a distribution
var ErrDistNotFound = errors.New("distribution not found")

// DistributionRepository provides a repository for manipulating CloudFront distributions to match desired configuration
type DistributionRepository interface {
	// ARNByGroup fetches the ARN from an existing Distribution in AWS that is owned by the operator and was created for
	// the given group.
	// Returns ErrDistNotFound if no existing Distribution was found.
	ARNByGroup(group string) (string, error)
	// Create creates the given Distribution on CloudFront. Returns the created dist.
	Create(Distribution) (Distribution, error)
	// Sync ensures the given Distribution is correctly configured on CloudFront. Returns synced dist.
	Sync(Distribution) (Distribution, error)
	// Delete deletes the Distribution at AWS
	Delete(Distribution) error
}

type repository struct {
	cloudfrontCli cloudfrontiface.CloudFrontAPI
	oacRepo       OACRepository
	taggingCli    resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	callerRef     CallerRefFn
	waitTimeout   time.Duration
}

// NewDistributionRepository creates a new AWS CloudFront DistributionRepository
func NewDistributionRepository(
	cloudFrontClient cloudfrontiface.CloudFrontAPI,
	taggingClient resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI,
	oacRepo OACRepository,
	callerRefFn CallerRefFn,
	waitTimeout time.Duration,
) DistributionRepository {
	return &repository{
		cloudfrontCli: cloudFrontClient,
		taggingCli:    taggingClient,
		oacRepo:       oacRepo,
		callerRef:     callerRefFn,
		waitTimeout:   waitTimeout,
	}
}

func (r repository) ARNByGroup(group string) (string, error) {
	input := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: []*string{aws.String("cloudfront:distribution")},
		TagFilters: []*resourcegroupstaggingapi.TagFilter{
			{
				Key:    aws.String(ownershipTagKey),
				Values: aws.StringSlice([]string{ownershipTagValue}),
			},
			{
				Key:    aws.String(groupTagKey),
				Values: aws.StringSlice([]string{group}),
			},
		},
	}

	out, err := r.taggingCli.GetResources(input)
	if err != nil {
		return "", fmt.Errorf("listing CloudFronts: %v", err)
	}

	if len(out.ResourceTagMappingList) == 0 {
		return "", ErrDistNotFound
	}

	if len(out.ResourceTagMappingList) > 1 {
		return "", fmt.Errorf("found more than one CloudFront with matching group tag (%s), state is inconsistent and can't continue", group)
	}

	return aws.StringValue(out.ResourceTagMappingList[0].ResourceARN), nil
}

func (r repository) Create(d Distribution) (Distribution, error) {
	config := newAWSDistributionConfig(d, r.callerRef)
	createInput := &awscloudfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &awscloudfront.DistributionConfigWithTags{
			DistributionConfig: config,
			Tags:               r.distributionTags(d),
		},
	}
	output, err := r.cloudfrontCli.CreateDistributionWithTags(createInput)
	if err != nil {
		return Distribution{}, fmt.Errorf("creating distribution: %v", err)
	}

	if err := r.createOACs(d); err != nil {
		return Distribution{}, fmt.Errorf("creating OACs: %v", err)
	}

	d.ID = *output.Distribution.Id
	d.Address = *output.Distribution.DomainName
	d.ARN = *output.Distribution.ARN
	return d, nil
}

func (r repository) Sync(d Distribution) (Distribution, error) {
	config := newAWSDistributionConfig(d, r.callerRef)
	output, err := r.distributionConfigByID(d.ID)
	if err != nil {
		return Distribution{}, fmt.Errorf("getting distribution config: %v", err)
	}

	syncedOACs, err := r.syncOACs(d, output)
	if err != nil {
		return Distribution{}, fmt.Errorf("syncing OACs: %v", err)
	}

	r.forEachOrigin(config, func(o *awscloudfront.Origin) {
		for _, oac := range syncedOACs {
			if aws.StringValue(o.Id) == oac.OriginName {
				o.SetOriginAccessControlId(oac.ID)
			}
		}
	})

	config.SetCallerReference(*output.DistributionConfig.CallerReference)
	config.SetDefaultRootObject(*output.DistributionConfig.DefaultRootObject)
	config.SetCustomErrorResponses(output.DistributionConfig.CustomErrorResponses)
	config.SetRestrictions(output.DistributionConfig.Restrictions)

	updateInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: config,
		IfMatch:            output.ETag,
		Id:                 aws.String(d.ID),
	}

	updateOut, err := r.cloudfrontCli.UpdateDistribution(updateInput)
	if err != nil {
		return Distribution{}, fmt.Errorf("updating distribution: %v", err)
	}

	tagsInput := &awscloudfront.TagResourceInput{
		Resource: aws.String(d.ARN),
		Tags:     r.distributionTags(d),
	}

	if _, err = r.cloudfrontCli.TagResource(tagsInput); err != nil {
		return Distribution{}, fmt.Errorf("updating tags: %v", err)
	}

	d.ID = *updateOut.Distribution.Id
	d.ARN = *updateOut.Distribution.ARN
	d.Address = *updateOut.Distribution.DomainName
	return d, nil
}

func (r repository) Delete(d Distribution) error {
	output, err := r.distributionConfigByID(d.ID)
	if err != nil {
		return cdnaws.IgnoreErrorCodef("getting distribution config: %v", err, awscloudfront.ErrCodeNoSuchDistribution)
	}

	if *output.DistributionConfig.Enabled {
		err = r.disableDist(output.DistributionConfig, d.ID, *output.ETag)
		if err != nil {
			return cdnaws.IgnoreErrorCodef("disabling distribution: %v", err, awscloudfront.ErrCodeNoSuchDistribution)
		}
	}

	eTag, err := r.waitUntilDeployed(d.ID)
	if err != nil {
		return cdnaws.IgnoreErrorCodef("waiting for distribution to be in deployed status: %w", err, awscloudfront.ErrCodeNoSuchDistribution)
	}

	input := &awscloudfront.DeleteDistributionInput{
		Id:      aws.String(d.ID),
		IfMatch: eTag,
	}
	_, err = r.cloudfrontCli.DeleteDistribution(input)
	if cdnaws.IgnoreErrorCode(err, awscloudfront.ErrCodeNoSuchDistribution) != nil {
		return err
	}

	if err := r.deleteOACs(output.DistributionConfig); err != nil {
		return fmt.Errorf("deleting OACs: %v", err)
	}

	return nil
}

func (r repository) distributionTags(d Distribution) *awscloudfront.Tags {
	var awsTags awscloudfront.Tags
	for k, v := range d.Tags {
		awsTags.Items = append(awsTags.Items, &awscloudfront.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	// map iteration is non-deterministic, so we sort the tags to make this deterministic and testable
	sort.Sort(byKey(awsTags.Items))
	return &awsTags
}

func (r repository) distributionConfigByID(id string) (*awscloudfront.GetDistributionConfigOutput, error) {
	input := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String(id),
	}
	output, err := r.cloudfrontCli.GetDistributionConfig(input)

	if err != nil {
		return nil, err
	}
	return output, nil
}

func (r repository) disableDist(config *awscloudfront.DistributionConfig, id, eTag string) error {
	config.Enabled = aws.Bool(false)
	updateInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: config,
		IfMatch:            aws.String(eTag),
		Id:                 aws.String(id),
	}

	_, err := r.cloudfrontCli.UpdateDistribution(updateInput)
	return err
}

const cfDeployedStatus = "Deployed"

func (r repository) waitUntilDeployed(id string) (*string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), (time.Second * 60))
	defer cancel()

	var eTag *string
	condition := func(ctx context.Context) (done bool, err error) {
		out, err := r.distributionByID(id)
		if err != nil {
			if cdnaws.IsErrorCode(err, awscloudfront.ErrCodeNoSuchDistribution) {
				return false, err
			}
			return false, nil
		}
		eTag = out.ETag
		return *out.Distribution.Status == cfDeployedStatus, nil
	}

	interval := time.Second * 10
	err := wait.PollUntilContextTimeout(ctx, interval, r.waitTimeout, true, condition)
	if err != nil {
		return nil, err
	}
	return eTag, nil
}

func (r repository) distributionByID(id string) (*awscloudfront.GetDistributionOutput, error) {
	input := &awscloudfront.GetDistributionInput{
		Id: aws.String(id),
	}
	return r.cloudfrontCli.GetDistribution(input)
}
func (r repository) createOACs(d Distribution) error {
	for _, o := range d.OACs() {
		if _, err := r.oacRepo.Sync(o); err != nil {
			return err
		}
	}
	return nil
}

func (r repository) syncOACs(desired Distribution, observed *awscloudfront.GetDistributionConfigOutput) (synced []OAC, err error) {
	toBeSynced, toBeDeleted := r.diffDesiredAndObservedOACs(desired, observed.DistributionConfig)

	for i, oac := range toBeSynced {
		syncedOAC, err := r.oacRepo.Sync(oac)
		if err != nil {
			return nil, err
		}
		toBeSynced[i] = syncedOAC
	}

	for _, oac := range toBeDeleted {
		_, err := r.oacRepo.Delete(oac)
		if err != nil {
			return nil, err
		}
	}

	return toBeSynced, nil
}

func (r repository) deleteOACs(cfg *awscloudfront.DistributionConfig) error {
	toBeDeleted := r.filterOACs(cfg, func(o *awscloudfront.Origin) bool {
		return !strhelper.IsEmptyOrNil(o.OriginAccessControlId)
	})

	for _, o := range toBeDeleted {
		if _, err := r.oacRepo.Delete(o); err != nil {
			return err
		}
	}

	return nil
}

func (r repository) diffDesiredAndObservedOACs(desired Distribution, observed *awscloudfront.DistributionConfig) (toBeSynced []OAC, toBeDeleted []OAC) {
	toBeSynced = desired.OACs()

	toBeDeleted = r.filterOACs(observed, func(o *awscloudfront.Origin) bool {
		originHasOAC := !strhelper.IsEmptyOrNil(o.OriginAccessControlId)
		originIsDesired := desired.HasOrigin(aws.StringValue(o.Id))

		return originHasOAC && !originIsDesired
	})

	return toBeSynced, toBeDeleted
}

func (r repository) filterOACs(cfg *awscloudfront.DistributionConfig, shouldInclude func(*awscloudfront.Origin) bool) []OAC {
	if !cfgHasOrigins(cfg) {
		return nil
	}

	var result []OAC
	for _, awsOrigin := range cfg.Origins.Items {
		if shouldInclude(awsOrigin) {
			result = append(result, OAC{
				ID: aws.StringValue(awsOrigin.OriginAccessControlId),
			})
		}
	}

	return result
}

func (r repository) forEachOrigin(cfg *awscloudfront.DistributionConfig, do func(*awscloudfront.Origin)) {
	if !cfgHasOrigins(cfg) {
		return
	}

	for _, o := range cfg.Origins.Items {
		do(o)
	}
}

func cfgHasOrigins(cfg *awscloudfront.DistributionConfig) bool {
	return cfg.Origins != nil && len(cfg.Origins.Items) > 0
}
