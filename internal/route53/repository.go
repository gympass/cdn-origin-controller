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
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"

	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	cfHostedZoneID         = "Z2FDTNDATAQYW2"
	cfEvaluateTargetHealth = false
	// ref: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ResourceRecordTypes.html
	numberOfSupportedRecordTypes = "13"
)

type filteredRecordSets struct {
	addressRecords []*route53.ResourceRecordSet
	txtRecord      *route53.ResourceRecordSet
}

// AliasRepository provides a layer to interact with the AWS API when manipulating Route53 records
type AliasRepository interface {
	// Upsert inserts or updates Aliases on Route53
	Upsert(aliases Aliases) error
	// Delete deletes Aliases on Route53
	Delete(aliases Aliases) error
}

type repository struct {
	awsClient route53iface.Route53API
}

// NewAliasRepository builds a new AliasRepository
func NewAliasRepository(awsClient route53iface.Route53API) AliasRepository {
	return &repository{awsClient: awsClient}
}

func (r repository) Upsert(aliases Aliases) error {
	if len(aliases.Entries) == 0 {
		return nil
	}

	var changes []*route53.Change
	for _, e := range aliases.Entries {
		existingRS, err := r.existingRecordSets(aliases.OwnershipTXTValue, aliases.HostedZoneID, e)
		if err != nil {
			return fmt.Errorf("fetching existing DNS records: %v", err)
		}

		var existingTXTRecords []*route53.ResourceRecord
		if existingRS.txtRecord != nil {
			existingTXTRecords = existingRS.txtRecord.ResourceRecords
		}

		changes = append(changes, r.newAliasChanges(aliases.Target, route53.ChangeActionUpsert, e)...)
		changes = append(changes, r.newTXTChangeForUpsert(aliases.OwnershipTXTValue, e.Name, existingTXTRecords...))
	}

	return r.requestChanges(changes, aliases.HostedZoneID, "Upserting Alias for CloudFront distribution managed by cdn-origin-controller")
}

func (r repository) Delete(aliases Aliases) error {
	if len(aliases.Entries) == 0 {
		return nil
	}

	target := aliases.Target

	var changes []*route53.Change
	for _, e := range aliases.Entries {
		recordSets, err := r.existingRecordSets(aliases.OwnershipTXTValue, aliases.HostedZoneID, e)
		if err != nil {
			return err
		}

		if recordSets.txtRecord == nil {
			return fmt.Errorf("ownership TXT record (%s) not found, can't delete address records", e.Name)
		}

		// we might not have a target present on aliases if the distribution was already deleted
		// if the record exists take it from there
		if len(recordSets.addressRecords) > 0 {
			target = *recordSets.addressRecords[0].AliasTarget.DNSName
		}

		changes = append(changes, r.newAliasChanges(target, route53.ChangeActionDelete, e)...)
		changes = append(changes, r.newTXTChangeForDelete(aliases.OwnershipTXTValue, e.Name, recordSets.txtRecord.ResourceRecords...))
	}

	return r.requestChanges(changes, aliases.HostedZoneID, "Deleting Alias for CloudFront distribution managed by cdn-origin-controller")
}

func (r repository) requestChanges(changes []*route53.Change, hostedZoneID, comment string) error {
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
			Comment: aws.String(comment),
		},
		HostedZoneId: aws.String(hostedZoneID),
	}

	_, err := r.awsClient.ChangeResourceRecordSets(input)
	return err
}

func (r repository) resourceRecordSetsByEntry(hostedZoneID string, entry Entry) ([]*route53.ResourceRecordSet, error) {
	sets, err := r.aliasResourceRecordsByEntry(hostedZoneID, entry)
	if err != nil {
		return nil, err
	}

	txtRS, err := r.txtResourceRecordSetByEntry(hostedZoneID, entry)
	if err != nil {
		return nil, err
	}

	if txtRS != nil {
		return append(sets, txtRS), nil
	}

	return sets, nil
}

func (r repository) aliasResourceRecordsByEntry(hostedZoneID string, entry Entry) ([]*route53.ResourceRecordSet, error) {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(hostedZoneID),
		StartRecordName: aws.String(entry.Name),
		MaxItems:        aws.String(numberOfSupportedRecordTypes),
	}

	output, err := r.awsClient.ListResourceRecordSets(input)
	if err != nil {
		return nil, err
	}

	return output.ResourceRecordSets, nil
}

func (r repository) txtResourceRecordSetByEntry(hostedZoneID string, entry Entry) (*route53.ResourceRecordSet, error) {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(hostedZoneID),
		StartRecordName: aws.String(entry.Name),
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
		if entry.Name == *rs.Name {
			if strhelper.Contains(entry.Types, *rs.Type) {
				filtered.addressRecords = append(filtered.addressRecords, rs)
			}
			if *rs.Type == route53.RRTypeTxt {
				filtered.txtRecord = rs
			}
		}
	}

	return filtered
}

func (r repository) existingRecordSets(ownershipTXTValue, hostedZoneID string, e Entry) (filteredRecordSets, error) {
	allRecordSets, err := r.resourceRecordSetsByEntry(hostedZoneID, e)
	if err != nil {
		return filteredRecordSets{}, err
	}

	recordSets := r.filterRecordSets(e, allRecordSets)

	if err := r.validateRecordSets(ownershipTXTValue, recordSets); err != nil {
		return filteredRecordSets{}, fmt.Errorf("validating records: %v", err)
	}
	return recordSets, nil
}

func (r repository) validateRecordSets(ownershipTXTValue string, filteredRs filteredRecordSets) error {
	if err := r.validateRoutingPolicies(filteredRs); err != nil {
		return err
	}

	if filteredRs.txtRecord == nil || !r.containsOwnershipRecord(filteredRs.txtRecord.ResourceRecords) {
		if len(filteredRs.addressRecords) > 0 {
			return errors.New("address record (A or AAAA) exists but is not managed by the controller")
		}
		return nil
	}

	err := r.validateOwnership(ownershipTXTValue, *filteredRs.txtRecord)
	if err != nil {
		return err
	}

	return nil
}

func (r repository) validateRoutingPolicies(sets filteredRecordSets) error {
	allRecords := append(sets.addressRecords, sets.txtRecord)
	for _, rs := range allRecords {
		if err := r.validateRoutingPolicy(rs); err != nil {
			return err
		}
	}
	return nil
}

func (r repository) validateRoutingPolicy(rs *route53.ResourceRecordSet) error {
	if rs == nil {
		return nil
	}

	if rs.Weight != nil {
		return fmt.Errorf("existing %s record (%q) has weighted routing policy. Routing policy should be simple", aws.StringValue(rs.Type), aws.StringValue(rs.Name))
	}

	if rs.GeoLocation != nil {
		return fmt.Errorf("existing %s record (%q) has geo-location routing policy. Routing policy should be simple", aws.StringValue(rs.Type), aws.StringValue(rs.Name))
	}

	if rs.CidrRoutingConfig != nil {
		return fmt.Errorf("existing %s record (%q) has ip-based routing policy. Routing policy should be simple", aws.StringValue(rs.Type), aws.StringValue(rs.Name))
	}

	return nil
}

func (r repository) validateOwnership(ownershipTXTValue string, rs route53.ResourceRecordSet) error {
	for _, rec := range rs.ResourceRecords {
		if r.isOwnedByThisClass(ownershipTXTValue, rec) {
			return nil
		}
		if r.isOwnedByDifferentClass(ownershipTXTValue, rec) {
			return fmt.Errorf("TXT record (%s) is managed by another CDN class (ownership value: %s)", *rs.Name, *rec.Value)
		}
	}

	return nil
}

func (r repository) isOwnedByDifferentClass(ownershipTXTValue string, rec *route53.ResourceRecord) bool {
	return !r.isOwnedByThisClass(ownershipTXTValue, rec) && r.isOwnershipRecord(rec)
}

func (r repository) isOwnedByThisClass(ownershipTXTValue string, rec *route53.ResourceRecord) bool {
	return *rec.Value == ownershipTXTValue
}

func (r repository) containsOwnershipRecord(records []*route53.ResourceRecord) bool {
	for _, rec := range records {
		if r.isOwnershipRecord(rec) {
			return true
		}
	}
	return false
}

func (r repository) isOwnershipRecord(record *route53.ResourceRecord) bool {
	return strings.Contains(*record.Value, txtOwnerKey)
}

func (r repository) newAliasChanges(target, action string, entry Entry) []*route53.Change {
	var changes []*route53.Change
	for _, rType := range entry.Types {
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

func (r repository) newTXTChangeForUpsert(ownershipTXTValue, name string, existingRecords ...*route53.ResourceRecord) *route53.Change {
	return r.newTXTChange(route53.ChangeActionUpsert, name, r.ensureTXTValue(ownershipTXTValue, existingRecords))
}

func (r repository) newTXTChangeForDelete(ownershipTXTValue, name string, existingRecords ...*route53.ResourceRecord) *route53.Change {
	action := route53.ChangeActionUpsert
	records := r.removeOwnershipRecord(ownershipTXTValue, existingRecords)
	if len(records) == 0 {
		action = route53.ChangeActionDelete
		records = existingRecords // the records must match the current ones for a successful delete
	}
	return r.newTXTChange(action, name, records)
}

func (r repository) newTXTChange(action, name string, records []*route53.ResourceRecord) *route53.Change {
	return &route53.Change{
		Action: aws.String(action),
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name:            aws.String(name),
			ResourceRecords: records,
			TTL:             aws.Int64(300),
			Type:            aws.String(route53.RRTypeTxt),
		},
	}
}

func (r repository) ensureTXTValue(ownershipTXTValue string, originalRecords []*route53.ResourceRecord) []*route53.ResourceRecord {
	for _, rec := range originalRecords {
		if *rec.Value == ownershipTXTValue {
			return originalRecords
		}
	}
	return append(originalRecords, &route53.ResourceRecord{Value: aws.String(ownershipTXTValue)})
}

func (r repository) removeOwnershipRecord(ownershipTXTValue string, originalRecords []*route53.ResourceRecord) []*route53.ResourceRecord {
	for i, rec := range originalRecords {
		if r.isOwnedByThisClass(ownershipTXTValue, rec) {
			return append(originalRecords[:i], originalRecords[i+1:]...)
		}
	}
	return originalRecords
}
