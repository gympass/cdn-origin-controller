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
)

var errNoSuchAOC = errors.New("aoc does not exist")

type AOCRepository interface {
	// Sync updates or creates a desired AOC. If successful, returns current AOC
	Sync(desired AOC) (AOC, error)
	// Delete deletes the AOC of given id. If successful, returns deleted AOC
	Delete(toBeDeleted AOC) (AOC, error)
}

func NewAOCRepository(client cloudfrontiface.CloudFrontAPI, aocLister AOCLister) AOCRepository {
	return aocRepository{client: client, aocLister: aocLister}
}

var _ AOCRepository = aocRepository{}

type aocRepository struct {
	client    cloudfrontiface.CloudFrontAPI
	aocLister AOCLister
}

func (r aocRepository) Sync(desired AOC) (AOC, error) {
	observed, err := r.getAOC(desired)
	if err == nil {
		return r.updateAOC(desired, observed)
	}

	if errors.Is(err, errNoSuchAOC) {
		return r.createAOC(desired)
	}

	return AOC{}, fmt.Errorf("fetching existing AOC: %v", err)
}

func (r aocRepository) Delete(toBeDeleted AOC) (AOC, error) {
	existingAOC, err := r.getAOC(toBeDeleted)
	if err != nil {
		return AOC{}, ignoreNoSuchAOC(err)
	}

	_, err = r.client.DeleteOriginAccessControl(&awscloudfront.DeleteOriginAccessControlInput{
		Id: aws.String(existingAOC.ID),
	})
	if cdnaws.IgnoreErrorCode(err, awscloudfront.ErrCodeNoSuchOriginAccessControl) != nil {
		return AOC{}, err
	}

	return existingAOC, nil
}

func (r aocRepository) createAOC(desired AOC) (AOC, error) {
	out, err := r.client.CreateOriginAccessControl(
		&awscloudfront.CreateOriginAccessControlInput{OriginAccessControlConfig: &awscloudfront.OriginAccessControlConfig{
			Description:                   aws.String(fmt.Sprintf("AOC for %s, managed by cdn-desired-controller", desired.OriginName)),
			Name:                          aws.String(desired.Name),
			OriginAccessControlOriginType: aws.String(awscloudfront.OriginAccessControlOriginTypesS3),
			SigningBehavior:               aws.String(awscloudfront.OriginAccessControlSigningBehaviorsAlways),
			SigningProtocol:               aws.String(awscloudfront.OriginAccessControlSigningProtocolsSigv4),
		}},
	)
	if err != nil {
		return AOC{}, err
	}

	aoc := out.OriginAccessControl.OriginAccessControlConfig
	return newAOCFromOriginAccessControlConfig(aoc, out.OriginAccessControl.Id, desired.OriginName), nil
}

func (r aocRepository) updateAOC(desired AOC, observed AOC) (AOC, error) {
	out, err := r.client.UpdateOriginAccessControl(
		&awscloudfront.UpdateOriginAccessControlInput{
			Id: aws.String(observed.ID),
			OriginAccessControlConfig: &awscloudfront.OriginAccessControlConfig{
				Description:                   aws.String(fmt.Sprintf("AOC for %s, managed by cdn-desired-controller", desired.OriginName)),
				Name:                          aws.String(desired.Name),
				OriginAccessControlOriginType: aws.String(awscloudfront.OriginAccessControlOriginTypesS3),
				SigningBehavior:               aws.String(awscloudfront.OriginAccessControlSigningBehaviorsAlways),
				SigningProtocol:               aws.String(awscloudfront.OriginAccessControlSigningProtocolsSigv4),
			}},
	)
	if err != nil {
		return AOC{}, err
	}

	aoc := out.OriginAccessControl.OriginAccessControlConfig
	return newAOCFromOriginAccessControlConfig(aoc, out.OriginAccessControl.Id, desired.OriginName), nil
}

func (r aocRepository) getAOC(aoc AOC) (AOC, error) {
	input := &awscloudfront.ListOriginAccessControlsInput{}

	var observed AOC
	found := false

	err := r.aocLister.ListOriginAccessControlsPages(input, func(output *awscloudfront.ListOriginAccessControlsOutput, lastPage bool) bool {
		for _, item := range output.OriginAccessControlList.Items {
			if aws.StringValue(item.Name) == aoc.Name {
				observed = newAOCFromOriginAccessControlSummary(item, aoc.OriginName)
				found = true
				return false
			}
		}
		return !lastPage
	})

	if err != nil {
		return AOC{}, fmt.Errorf("listing AOCs: %v", err)
	}

	if !found {
		return AOC{}, errNoSuchAOC
	}

	return observed, nil
}

func newAOCFromOriginAccessControlConfig(cfg *awscloudfront.OriginAccessControlConfig, id *string, originName string) AOC {
	return AOC{
		ID:                            aws.StringValue(id),
		Name:                          aws.StringValue(cfg.Name),
		OriginName:                    originName,
		OriginAccessControlOriginType: aws.StringValue(cfg.OriginAccessControlOriginType),
		SigningBehavior:               aws.StringValue(cfg.SigningBehavior),
		SigningProtocol:               aws.StringValue(cfg.SigningProtocol),
	}
}

func newAOCFromOriginAccessControlSummary(summary *awscloudfront.OriginAccessControlSummary, originName string) AOC {
	return AOC{
		ID:                            aws.StringValue(summary.Id),
		Name:                          aws.StringValue(summary.Name),
		OriginName:                    originName,
		OriginAccessControlOriginType: aws.StringValue(summary.OriginAccessControlOriginType),
		SigningBehavior:               aws.StringValue(summary.SigningBehavior),
		SigningProtocol:               aws.StringValue(summary.SigningProtocol),
	}
}

func ignoreNoSuchAOC(err error) error {
	if errors.Is(err, errNoSuchAOC) {
		return nil
	}
	return err
}
