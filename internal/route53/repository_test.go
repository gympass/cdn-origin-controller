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

package route53_test

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awsroute53 "github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/route53"
)

const (
	cfHostedZoneID               = "Z2FDTNDATAQYW2"
	numberOfSupportedRecordTypes = "13"
	txtOwnerKey                  = "cdn-origin-controller/owner"
)

type awsClientMock struct {
	mock.Mock
	route53iface.Route53API

	ExpectedListRSSOutForTXTRecord      *awsroute53.ListResourceRecordSetsOutput
	ExpectedListRRSOutForAddressRecords *awsroute53.ListResourceRecordSetsOutput
}

func (m *awsClientMock) ChangeResourceRecordSets(in *awsroute53.ChangeResourceRecordSetsInput) (*awsroute53.ChangeResourceRecordSetsOutput, error) {
	args := m.Called(in)
	return nil, args.Error(0) // we discard the out and only care about the error, no need to mock it
}

func (m *awsClientMock) ListResourceRecordSets(in *awsroute53.ListResourceRecordSetsInput) (*awsroute53.ListResourceRecordSetsOutput, error) {
	args := m.Called(in)

	result := m.ExpectedListRSSOutForTXTRecord
	if in.StartRecordType == nil {
		result = m.ExpectedListRRSOutForAddressRecords
	}

	return result, args.Error(0)
}

func TestRunAliasRepositoryTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &AliasRepositoryTestSuite{})
}

type AliasRepositoryTestSuite struct {
	suite.Suite
}

func (s *AliasRepositoryTestSuite) TestUpsert_NoEntriesOnAliases() {
	r := route53.NewAliasRepository(&awsClientMock{}, config.Config{})
	a := route53.NewAliases("target.foo.bar.", nil, false)
	s.NoError(r.Upsert(a))
}

func (s *AliasRepositoryTestSuite) TestUpsert_FailureListingAddressRecords() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(errors.New("mock err")).Once()

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	err := repo.Upsert(aliases)
	s.Error(err)
	s.Contains(err.Error(), "mock err")
}

func (s *AliasRepositoryTestSuite) TestUpsert_FailureListingTXTRecord() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: nil, // we're mocking when there are no existing records
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(errors.New("mock err")).Once()

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	err := repo.Upsert(aliases)
	s.Error(err)
	s.Contains(err.Error(), "mock err")
}

func (s *AliasRepositoryTestSuite) TestUpsert_EntriesDontExist() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: nil, // we're mocking when there are no existing records
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: nil, // we're mocking when there are no existing records
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	expectedChangeRRSInput := &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String("zone id"),
		ChangeBatch: &awsroute53.ChangeBatch{
			Changes: []*awsroute53.Change{
				{ // A record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionUpsert),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeA),
						AliasTarget: &awsroute53.AliasTarget{
							DNSName:              aws.String("target.foo.bar."),
							EvaluateTargetHealth: aws.Bool(false),
							HostedZoneId:         aws.String(cfHostedZoneID),
						},
					},
				},
				{ // TXT record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionUpsert),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeTxt),
						TTL:  aws.Int64(300),
						ResourceRecords: []*awsroute53.ResourceRecord{
							{
								Value: aws.String(`"cdn-origin-controller/owner=owner value"`),
							},
						},
					},
				},
			},
			Comment: aws.String("Upserting Alias for CloudFront distribution managed by cdn-origin-controller"),
		},
	}
	mockClient.On("ChangeResourceRecordSets", expectedChangeRRSInput).Return(noError).Once()

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	s.NoError(repo.Upsert(aliases))
}

func (s *AliasRepositoryTestSuite) TestUpsert_TXTExists_HasOtherValues() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: nil, // we're mocking when there are no existing records
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeTxt),
				TTL:  aws.Int64(300),
				ResourceRecords: []*awsroute53.ResourceRecord{
					{
						Value: aws.String("some other value"),
					},
				},
			},
		},
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	expectedChangeRRSInput := &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String("zone id"),
		ChangeBatch: &awsroute53.ChangeBatch{
			Changes: []*awsroute53.Change{
				{ // A record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionUpsert),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeA),
						AliasTarget: &awsroute53.AliasTarget{
							DNSName:              aws.String("target.foo.bar."),
							EvaluateTargetHealth: aws.Bool(false),
							HostedZoneId:         aws.String(cfHostedZoneID),
						},
					},
				},
				{ // TXT record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionUpsert),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeTxt),
						TTL:  aws.Int64(300),
						ResourceRecords: []*awsroute53.ResourceRecord{
							{
								Value: aws.String("some other value"),
							},
							{
								Value: aws.String(`"cdn-origin-controller/owner=owner value"`),
							},
						},
					},
				},
			},
			Comment: aws.String("Upserting Alias for CloudFront distribution managed by cdn-origin-controller"),
		},
	}
	mockClient.On("ChangeResourceRecordSets", expectedChangeRRSInput).Return(noError).Once()

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	s.NoError(repo.Upsert(aliases))
}

func (s *AliasRepositoryTestSuite) TestUpsert_AddressRecordsExist_OwnedByNoClass() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeA),
				TTL:  aws.Int64(300),
				ResourceRecords: []*awsroute53.ResourceRecord{
					{
						Value: aws.String("10.0.0.1"),
					},
				},
			},
		},
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: nil, // we're mocking when there's no ownership record
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	err := repo.Upsert(aliases)
	s.Error(err)
	s.Contains(err.Error(), "address record (A or AAAA) exists but is not managed by the controller")
}

func (s *AliasRepositoryTestSuite) TestUpsert_RecordsExist_OwnedByAnotherClass() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeA),
				TTL:  aws.Int64(300),
				AliasTarget: &awsroute53.AliasTarget{
					DNSName:              aws.String("another-target.foo.bar."),
					EvaluateTargetHealth: aws.Bool(false),
					HostedZoneId:         aws.String(cfHostedZoneID),
				},
			},
		},
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeTxt),
				TTL:  aws.Int64(300),
				ResourceRecords: []*awsroute53.ResourceRecord{
					{
						Value: aws.String(txtOwnerKey + "=another owner class"),
					},
				},
			},
		},
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	err := repo.Upsert(aliases)
	s.Error(err)
	s.Contains(err.Error(), "is managed by another CDN class")
}

func (s *AliasRepositoryTestSuite) TestUpsert_RecordsExist_AlreadyOwnedByThisClass() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeA),
				TTL:  aws.Int64(300),
				AliasTarget: &awsroute53.AliasTarget{
					DNSName:              aws.String("target.foo.bar."),
					EvaluateTargetHealth: aws.Bool(false),
					HostedZoneId:         aws.String(cfHostedZoneID),
				},
			},
		},
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeTxt),
				TTL:  aws.Int64(300),
				ResourceRecords: []*awsroute53.ResourceRecord{
					{
						Value: aws.String("some other value"),
					},
					{
						Value: aws.String(`"cdn-origin-controller/owner=owner value"`),
					},
				},
			},
		},
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	expectedChangeRRSInput := &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String("zone id"),
		ChangeBatch: &awsroute53.ChangeBatch{
			Changes: []*awsroute53.Change{
				{ // A record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionUpsert),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeA),
						AliasTarget: &awsroute53.AliasTarget{
							DNSName:              aws.String("target.foo.bar."),
							EvaluateTargetHealth: aws.Bool(false),
							HostedZoneId:         aws.String(cfHostedZoneID),
						},
					},
				},
				{ // TXT record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionUpsert),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeTxt),
						TTL:  aws.Int64(300),
						ResourceRecords: []*awsroute53.ResourceRecord{
							{
								Value: aws.String("some other value"),
							},
							{
								Value: aws.String(`"cdn-origin-controller/owner=owner value"`),
							},
						},
					},
				},
			},
			Comment: aws.String("Upserting Alias for CloudFront distribution managed by cdn-origin-controller"),
		},
	}
	mockClient.On("ChangeResourceRecordSets", expectedChangeRRSInput).Return(noError).Once()

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	s.NoError(repo.Upsert(aliases))
}

func (s *AliasRepositoryTestSuite) TestDelete_NoEntriesOnAliases() {
	r := route53.NewAliasRepository(&awsClientMock{}, config.Config{})
	a := route53.NewAliases("target.foo.bar.", nil, false)
	s.NoError(r.Delete(a))
}

func (s *AliasRepositoryTestSuite) TestDelete_FailureListingRecords() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(errors.New("mock err")).Once()

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	err := repo.Delete(aliases)
	s.Error(err)
	s.Contains(err.Error(), "mock err")
}

func (s *AliasRepositoryTestSuite) TestDelete_RecordsDontExist() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: nil,
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: nil,
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	s.EqualError(repo.Delete(aliases), "ownership TXT record (alias.foo.bar.) not found, can't delete address records")
}

func (s *AliasRepositoryTestSuite) TestDelete_RecordsExist_TXTHasNoAdditionalValues() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeA),
				TTL:  aws.Int64(300),
				AliasTarget: &awsroute53.AliasTarget{
					DNSName:              aws.String("target.foo.bar."),
					EvaluateTargetHealth: aws.Bool(false),
					HostedZoneId:         aws.String(cfHostedZoneID),
				},
			},
		},
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeTxt),
				TTL:  aws.Int64(300),
				ResourceRecords: []*awsroute53.ResourceRecord{
					{
						Value: aws.String(`"cdn-origin-controller/owner=owner value"`),
					},
				},
			},
		},
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	expectedChangeRRSInput := &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String("zone id"),
		ChangeBatch: &awsroute53.ChangeBatch{
			Changes: []*awsroute53.Change{
				{ // A record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionDelete),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeA),
						AliasTarget: &awsroute53.AliasTarget{
							DNSName:              aws.String("target.foo.bar."),
							EvaluateTargetHealth: aws.Bool(false),
							HostedZoneId:         aws.String(cfHostedZoneID),
						},
					},
				},
				{ // TXT record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionDelete),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeTxt),
						TTL:  aws.Int64(300),
						ResourceRecords: []*awsroute53.ResourceRecord{
							{
								Value: aws.String(`"cdn-origin-controller/owner=owner value"`),
							},
						},
					},
				},
			},
			Comment: aws.String("Deleting Alias for CloudFront distribution managed by cdn-origin-controller"),
		},
	}
	mockClient.On("ChangeResourceRecordSets", expectedChangeRRSInput).Return(noError).Once()

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	s.NoError(repo.Delete(aliases))
}

func (s *AliasRepositoryTestSuite) TestDelete_RecordsExist_TXTHasAdditionalValues() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeA),
				TTL:  aws.Int64(300),
				AliasTarget: &awsroute53.AliasTarget{
					DNSName:              aws.String("target.foo.bar."),
					EvaluateTargetHealth: aws.Bool(false),
					HostedZoneId:         aws.String(cfHostedZoneID),
				},
			},
		},
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeTxt),
				TTL:  aws.Int64(300),
				ResourceRecords: []*awsroute53.ResourceRecord{
					{
						Value: aws.String("some other value"),
					},
					{
						Value: aws.String(`"cdn-origin-controller/owner=owner value"`),
					},
				},
			},
		},
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	expectedChangeRRSInput := &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String("zone id"),
		ChangeBatch: &awsroute53.ChangeBatch{
			Changes: []*awsroute53.Change{
				{ // A record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionDelete),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeA),
						AliasTarget: &awsroute53.AliasTarget{
							DNSName:              aws.String("target.foo.bar."),
							EvaluateTargetHealth: aws.Bool(false),
							HostedZoneId:         aws.String(cfHostedZoneID),
						},
					},
				},
				{ // TXT record for alias.foo.bar.
					Action: aws.String(awsroute53.ChangeActionUpsert),
					ResourceRecordSet: &awsroute53.ResourceRecordSet{
						Name: aws.String("alias.foo.bar."),
						Type: aws.String(awsroute53.RRTypeTxt),
						TTL:  aws.Int64(300),
						ResourceRecords: []*awsroute53.ResourceRecord{
							{
								Value: aws.String("some other value"),
							},
						},
					},
				},
			},
			Comment: aws.String("Deleting Alias for CloudFront distribution managed by cdn-origin-controller"),
		},
	}
	mockClient.On("ChangeResourceRecordSets", expectedChangeRRSInput).Return(noError).Once()

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	s.NoError(repo.Delete(aliases))
}

func (s *AliasRepositoryTestSuite) TestDelete_RecordsExist_TXTNotOwnedByThisClass() {
	cfg := config.Config{
		CloudFrontRoute53TxtOwnerValue: "owner value",
		CloudFrontRoute53HostedZoneID:  "zone id",
	}
	mockClient := &awsClientMock{}

	expectedListRRSInputForAddresses := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}
	expectedListRRSOutputForAddresses := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeA),
				TTL:  aws.Int64(300),
				AliasTarget: &awsroute53.AliasTarget{
					DNSName:              aws.String("another-target.foo.bar."),
					EvaluateTargetHealth: aws.Bool(false),
					HostedZoneId:         aws.String(cfHostedZoneID),
				},
			},
		},
	}
	var noError error
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForAddresses).Return(noError).Once()
	mockClient.ExpectedListRRSOutForAddressRecords = expectedListRRSOutputForAddresses

	expectedListRRSInputForTXT := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cfg.CloudFrontRoute53HostedZoneID),
		StartRecordName: aws.String("alias.foo.bar."),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(awsroute53.RRTypeTxt),
	}
	expectedListRRSOutputForTXT := &awsroute53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*awsroute53.ResourceRecordSet{
			{
				Name: aws.String("alias.foo.bar."),
				Type: aws.String(awsroute53.RRTypeTxt),
				TTL:  aws.Int64(300),
				ResourceRecords: []*awsroute53.ResourceRecord{
					{
						Value: aws.String(txtOwnerKey + "=another owner class"),
					},
				},
			},
		},
	}
	mockClient.On("ListResourceRecordSets", expectedListRRSInputForTXT).Return(noError).Once()
	mockClient.ExpectedListRSSOutForTXTRecord = expectedListRRSOutputForTXT

	repo := route53.NewAliasRepository(mockClient, cfg)
	aliases := route53.NewAliases("target.foo.bar.", []string{"alias.foo.bar."}, false)
	err := repo.Delete(aliases)
	s.Error(err)
	s.Contains(err.Error(), "is managed by another CDN class")
}
