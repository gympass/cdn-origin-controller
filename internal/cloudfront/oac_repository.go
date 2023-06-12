// Copyright (c) 2023 GPBR Participacoes LTDA.
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
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"

	cdnaws "github.com/Gympass/cdn-origin-controller/internal/aws"
	"github.com/Gympass/cdn-origin-controller/internal/config"
)

var errNoSuchOAC = errors.New("oac does not exist")

type OACRepository interface {
	// Sync updates or creates a desired OAC. If successful, returns current OAC
	Sync(desired OAC) (OAC, error)
	// Delete deletes the OAC of given id. If successful, returns deleted OAC
	Delete(toBeDeleted OAC) (OAC, error)
}

func NewOACRepository(client cloudfrontiface.CloudFrontAPI, oacLister OACLister, cfg config.Config) OACRepository {
	return oacRepository{client: client, oacLister: oacLister, cfg: cfg}
}

var _ OACRepository = oacRepository{}

type oacRepository struct {
	client    cloudfrontiface.CloudFrontAPI
	oacLister OACLister
	cfg       config.Config
}

func (r oacRepository) Sync(desired OAC) (OAC, error) {
	observed, eTag, err := r.getOAC(desired)
	if err == nil {
		return r.updateOAC(desired, observed, eTag)
	}

	if errors.Is(err, errNoSuchOAC) {
		return r.createOAC(desired)
	}

	return OAC{}, fmt.Errorf("fetching existing OAC: %v", err)
}

func (r oacRepository) Delete(toBeDeleted OAC) (OAC, error) {
	if !r.cfg.DeletionEnabled {
		return OAC{}, nil
	}

	existingOAC, eTag, err := r.getOAC(toBeDeleted)
	if err != nil {
		return OAC{}, ignoreNoSuchOAC(err)
	}

	_, err = r.client.DeleteOriginAccessControl(&awscloudfront.DeleteOriginAccessControlInput{
		Id:      aws.String(existingOAC.ID),
		IfMatch: eTag,
	})
	if cdnaws.IgnoreErrorCode(err, awscloudfront.ErrCodeNoSuchOriginAccessControl) != nil {
		return OAC{}, err
	}

	return existingOAC, nil
}

func (r oacRepository) createOAC(desired OAC) (OAC, error) {
	out, err := r.client.CreateOriginAccessControl(
		&awscloudfront.CreateOriginAccessControlInput{OriginAccessControlConfig: &awscloudfront.OriginAccessControlConfig{
			Description:                   aws.String(r.oacDescription(desired)),
			Name:                          aws.String(desired.Name),
			OriginAccessControlOriginType: aws.String(awscloudfront.OriginAccessControlOriginTypesS3),
			SigningBehavior:               aws.String(awscloudfront.OriginAccessControlSigningBehaviorsAlways),
			SigningProtocol:               aws.String(awscloudfront.OriginAccessControlSigningProtocolsSigv4),
		}},
	)
	if err != nil {
		return OAC{}, err
	}

	oac := out.OriginAccessControl.OriginAccessControlConfig
	return newOACFromOriginAccessControlConfig(oac, out.OriginAccessControl.Id, desired.OriginName), nil
}

func (r oacRepository) updateOAC(desired, observed OAC, eTag *string) (OAC, error) {
	out, err := r.client.UpdateOriginAccessControl(
		&awscloudfront.UpdateOriginAccessControlInput{
			Id:      aws.String(observed.ID),
			IfMatch: eTag,
			OriginAccessControlConfig: &awscloudfront.OriginAccessControlConfig{
				Description:                   aws.String(r.oacDescription(desired)),
				Name:                          aws.String(desired.Name),
				OriginAccessControlOriginType: aws.String(awscloudfront.OriginAccessControlOriginTypesS3),
				SigningBehavior:               aws.String(awscloudfront.OriginAccessControlSigningBehaviorsAlways),
				SigningProtocol:               aws.String(awscloudfront.OriginAccessControlSigningProtocolsSigv4),
			}},
	)
	if err != nil {
		return OAC{}, err
	}

	oac := out.OriginAccessControl.OriginAccessControlConfig
	return newOACFromOriginAccessControlConfig(oac, out.OriginAccessControl.Id, desired.OriginName), nil
}

func (r oacRepository) oacDescription(desired OAC) string {
	return fmt.Sprintf("OAC for %s, managed by cdn-origin-controller", desired.OriginName)
}

func (r oacRepository) getOAC(oac OAC) (observed OAC, eTag *string, err error) {
	input := &awscloudfront.ListOriginAccessControlsInput{}

	found := false
	err = r.oacLister.ListOriginAccessControlsPages(input, func(output *awscloudfront.ListOriginAccessControlsOutput, lastPage bool) bool {
		for _, item := range output.OriginAccessControlList.Items {
			if aws.StringValue(item.Id) == oac.ID || aws.StringValue(item.Name) == oac.Name {
				observed = newOACFromOriginAccessControlSummary(item, oac.OriginName)
				found = true
				return false
			}
		}
		return !lastPage
	})

	if err != nil {
		return OAC{}, nil, fmt.Errorf("listing OACs: %v", err)
	}

	if !found {
		return OAC{}, nil, errNoSuchOAC
	}

	// Listing returns no eTag, so we need to fetch again, but directly this time.
	// Additionally, we don't care about what was previously configured in the OAC,
	// so we can just use whatever eTag is current: we don't care if the OAC was
	// updated since we saw it last, we want to overwrite the entire thing.
	eTag, err = r.getETag(observed.ID)
	return observed, eTag, err
}

func (r oacRepository) getETag(id string) (eTag *string, err error) {
	out, err := r.client.GetOriginAccessControl(&awscloudfront.GetOriginAccessControlInput{
		Id: aws.String(id),
	})
	if err != nil {
		return nil, err
	}

	return out.ETag, nil
}

func newOACFromOriginAccessControlConfig(cfg *awscloudfront.OriginAccessControlConfig, id *string, originName string) OAC {
	return OAC{
		ID:                            aws.StringValue(id),
		Name:                          aws.StringValue(cfg.Name),
		OriginName:                    originName,
		OriginAccessControlOriginType: aws.StringValue(cfg.OriginAccessControlOriginType),
		SigningBehavior:               aws.StringValue(cfg.SigningBehavior),
		SigningProtocol:               aws.StringValue(cfg.SigningProtocol),
	}
}

func newOACFromOriginAccessControlSummary(summary *awscloudfront.OriginAccessControlSummary, originName string) OAC {
	return OAC{
		ID:                            aws.StringValue(summary.Id),
		Name:                          aws.StringValue(summary.Name),
		OriginName:                    originName,
		OriginAccessControlOriginType: aws.StringValue(summary.OriginAccessControlOriginType),
		SigningBehavior:               aws.StringValue(summary.SigningBehavior),
		SigningProtocol:               aws.StringValue(summary.SigningProtocol),
	}
}

func ignoreNoSuchOAC(err error) error {
	if errors.Is(err, errNoSuchOAC) {
		return nil
	}
	return err
}
