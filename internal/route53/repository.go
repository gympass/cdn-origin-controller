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

package route53

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	cfHostedZoneID         = "Z2FDTNDATAQYW2"
	cfEvaluateTargetHealth = false
	txtOwnerKey            = "cdn-origin-controller/owner"
	txtPrefix              = "cdn-origin-controller-"
	// ref: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ResourceRecordTypes.html
	numberOfSupportedRecordTypes = "13"
)

type filteredRecordSets struct {
	addressRecords  []*route53.ResourceRecordSet
	ownershipRecord *route53.ResourceRecordSet
}

// AliasRepository provides a layer to interact with the AWS API when manipulating Route53 records
type AliasRepository interface {
	// Upsert inserts or updates Aliases on Route53
	Upsert(aliases Aliases) error
	// Delete deletes Aliases on Route53
	Delete(aliases Aliases) error
}

type repository struct {
	awsClient         route53iface.Route53API
	hostedZoneID      string
	ownershipTXTValue string
}

// NewRoute53AliasRepository builds a new AliasRepository
func NewRoute53AliasRepository(awsClient route53iface.Route53API, config config.Config) AliasRepository {
	txtValue := fmt.Sprintf(`"%s=%s"`, txtOwnerKey, config.CloudFrontRoute53TxtOwnerValue)
	return &repository{awsClient: awsClient, hostedZoneID: config.CloudFrontRoute53HostedZoneID, ownershipTXTValue: txtValue}
}

func (r repository) Upsert(aliases Aliases) error {
	if len(aliases.Entries) == 0 {
		return nil
	}

	var changes []*route53.Change

	for _, e := range aliases.Entries {
		allRecordSets, err := r.resourceRecordSetsByEntry(e)
		if err != nil {
			return err
		}

		recordSets := r.filterRecordSets(e, allRecordSets)

		if err := r.validate(recordSets); err != nil {
			return fmt.Errorf("validating records: %v", err)
		}

		changes = append(changes, r.newAliasChanges(aliases.Target, route53.ChangeActionUpsert, e)...)
		changes = append(changes, r.newTXTChange(route53.ChangeActionUpsert, e.Name))
	}

	return r.requestChanges(changes, "CloudFront distribution managed by cdn-origin-controller")
}

func (r repository) Delete(aliases Aliases) error {
	if len(aliases.Entries) == 0 {
		return nil
	}
	var changes []*route53.Change

	for _, e := range aliases.Entries {
		changes = append(changes, r.newAliasChanges(aliases.Target, route53.ChangeActionDelete, e)...)
		changes = append(changes, r.newTXTChange(route53.ChangeActionDelete, e.Name))
	}

	return r.requestChanges(changes, "Deleting Alias for CloudFront distribution managed by cdn-origin-controller")
}

func (r repository) requestChanges(changes []*route53.Change, comment string) error {
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
			Comment: aws.String(comment),
		},
		HostedZoneId: aws.String(r.hostedZoneID),
	}

	_, err := r.awsClient.ChangeResourceRecordSets(input)
	return err
}

func (r repository) resourceRecordSetsByEntry(entry Entry) ([]*route53.ResourceRecordSet, error) {
	sets, err := r.aliasResourceRecordsByEntry(entry)
	if err != nil {
		return nil, err
	}

	txtRS, err := r.txtRecordSetByEntry(entry)
	if err != nil {
		return nil, err
	}

	return append(sets, txtRS), nil
}

func (r repository) aliasResourceRecordsByEntry(entry Entry) ([]*route53.ResourceRecordSet, error) {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(r.hostedZoneID),
		StartRecordName: aws.String(entry.Name),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}

	output, err := r.awsClient.ListResourceRecordSets(input)
	if err != nil {
		return nil, err
	}

	return output.ResourceRecordSets, nil
}

func (r repository) txtRecordSetByEntry(entry Entry) (*route53.ResourceRecordSet, error) {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(r.hostedZoneID),
		StartRecordName: aws.String(txtName(entry.Name)),
		MaxItems:        aws.String("1"),
		StartRecordType: aws.String(route53.RRTypeTxt),
	}

	output, err := r.awsClient.ListResourceRecordSets(input)
	if err != nil {
		return nil, err
	}

	if len(output.ResourceRecordSets) > 0 {
		return output.ResourceRecordSets[0], nil
	}

	return nil, nil
}

func (r repository) filterRecordSets(entry Entry, recordSets []*route53.ResourceRecordSet) filteredRecordSets {
	filtered := filteredRecordSets{}

	for _, rs := range recordSets {
		if entry.Name == *rs.Name && strhelper.Contains(entry.Type, *rs.Type) {
			filtered.addressRecords = append(filtered.addressRecords, rs)
		}
		if *rs.Type == route53.RRTypeTxt && txtName(entry.Name) == *rs.Name {
			filtered.ownershipRecord = rs
		}
	}

	return filtered
}

func (r repository) validate(filteredRs filteredRecordSets) error {
	if filteredRs.ownershipRecord == nil {
		if len(filteredRs.addressRecords) > 0 {
			return errors.New("address record (A or AAAA) exists but is not managed by the controller")
		}
		return nil
	}

	hasOwnerValue := r.ownershipTXTValue == *filteredRs.ownershipRecord.ResourceRecords[0].Value
	if !hasOwnerValue {
		return fmt.Errorf("TXT record (%s) already exists, but is not managed by the controller", *filteredRs.ownershipRecord.Name)
	}

	return nil
}

func (r repository) newAliasChanges(target, action string, entry Entry) []*route53.Change {
	var changes []*route53.Change
	for _, rType := range entry.Type {
		changes = append(changes, r.newAliasChange(target, action, entry.Name, rType))
	}
	return changes
}

func (r repository) newAliasChange(target, action, name, rType string) *route53.Change {
	return &route53.Change{
		Action: aws.String(action),
		ResourceRecordSet: &route53.ResourceRecordSet{
			AliasTarget: &route53.AliasTarget{
				DNSName:              aws.String(target),
				EvaluateTargetHealth: aws.Bool(cfEvaluateTargetHealth),
				HostedZoneId:         aws.String(cfHostedZoneID),
			},
			Name: aws.String(name),
			Type: aws.String(rType),
		},
	}
}

func (r repository) newTXTChange(action, name string) *route53.Change {
	return &route53.Change{
		Action: aws.String(action),
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name:            aws.String(txtName(name)),
			ResourceRecords: []*route53.ResourceRecord{{Value: aws.String(r.ownershipTXTValue)}},
			TTL:             aws.Int64(300),
			Type:            aws.String(route53.RRTypeTxt),
		},
	}
}

func txtName(originalName string) string {
	return txtPrefix + originalName
}
