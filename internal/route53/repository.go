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

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	cfHostedZoneID         = "Z2FDTNDATAQYW2"
	cfEvaluateTargetHealth = false
	txtOwnerKey            = "cdn-origin-controller/owner"
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
		recordSets, err := r.recordSets(e)
		if err != nil {
			return fmt.Errorf("generating filtered record sets: %v", err)
		}

		var existingRecords []*route53.ResourceRecord
		if recordSets.txtRecord != nil {
			existingRecords = recordSets.txtRecord.ResourceRecords
		}

		changes = append(changes, r.newAliasChanges(aliases.Target, route53.ChangeActionUpsert, e)...)
		changes = append(changes, r.newTXTChangeForUpsert(e.Name, existingRecords...))
	}

	return r.requestChanges(changes, "CloudFront distribution managed by cdn-origin-controller")
}

func (r repository) Delete(aliases Aliases) error {
	if len(aliases.Entries) == 0 {
		return nil
	}
	var changes []*route53.Change

	for _, e := range aliases.Entries {
		recordSets, err := r.recordSets(e)
		if err != nil {
			return err
		}

		if recordSets.txtRecord == nil {
			return fmt.Errorf("ownership TXT record (%s) not found, can't delete address records", e.Name)
		}

		changes = append(changes, r.newAliasChanges(aliases.Target, route53.ChangeActionDelete, e)...)
		changes = append(changes, r.newTXTChangeForDelete(e.Name, recordSets.txtRecord.ResourceRecords...))
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

	txtRS, err := r.txtResourceRecordSetByEntry(entry)
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

func (r repository) txtResourceRecordSetByEntry(entry Entry) (*route53.ResourceRecordSet, error) {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(r.hostedZoneID),
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
			if strhelper.Contains(entry.Type, *rs.Type) {
				filtered.addressRecords = append(filtered.addressRecords, rs)
			}
			if *rs.Type == route53.RRTypeTxt {
				filtered.txtRecord = rs
			}
		}
	}

	return filtered
}

func (r repository) recordSets(e Entry) (filteredRecordSets, error) {
	allRecordSets, err := r.resourceRecordSetsByEntry(e)
	if err != nil {
		return filteredRecordSets{}, err
	}

	recordSets := r.filterRecordSets(e, allRecordSets)

	if err := r.validateRecordSets(recordSets); err != nil {
		return filteredRecordSets{}, fmt.Errorf("validating records: %v", err)
	}
	return recordSets, nil
}

func (r repository) validateRecordSets(filteredRs filteredRecordSets) error {
	if filteredRs.txtRecord == nil || !r.containsOwnershipRecord(filteredRs.txtRecord.ResourceRecords) {
		if len(filteredRs.addressRecords) > 0 {
			return errors.New("address record (A or AAAA) exists but is not managed by the controller")
		}
		return nil
	}

	err := r.validateOwnership(*filteredRs.txtRecord)
	if err != nil {
		return err
	}

	return nil
}

func (r repository) validateOwnership(rs route53.ResourceRecordSet) error {
	var found bool
	for _, rec := range rs.ResourceRecords {
		if r.isOwnedByThisClass(rec) {
			found = true
		}
		if r.isOwnedByDifferentClass(rec) {
			return fmt.Errorf("TXT record (%s) is managed by another CDN class (ownership value: %s)", *rs.Name, *rec.Value)
		}
	}

	if !found {
		return fmt.Errorf("TXT record (%s) is not managed by this CDN class", *rs.Name)
	}

	return nil
}

func (r repository) isOwnedByDifferentClass(rec *route53.ResourceRecord) bool {
	return !r.isOwnedByThisClass(rec) && r.isOwnershipRecord(rec)
}

func (r repository) isOwnedByThisClass(rec *route53.ResourceRecord) bool {
	return *rec.Value == r.ownershipTXTValue
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

func (r repository) newTXTChangeForUpsert(name string, existingRecords ...*route53.ResourceRecord) *route53.Change {
	return r.newTXTChange(route53.ChangeActionUpsert, name, r.ensureTXTValue(existingRecords))
}

func (r repository) newTXTChangeForDelete(name string, existingRecords ...*route53.ResourceRecord) *route53.Change {
	action := route53.ChangeActionUpsert
	records := r.removeOwnershipRecord(existingRecords)
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

func (r repository) ensureTXTValue(originalRecords []*route53.ResourceRecord) []*route53.ResourceRecord {
	for _, rec := range originalRecords {
		if *rec.Value == r.ownershipTXTValue {
			return originalRecords
		}
	}
	return append(originalRecords, &route53.ResourceRecord{Value: aws.String(r.ownershipTXTValue)})
}

func (r repository) removeOwnershipRecord(originalRecords []*route53.ResourceRecord) []*route53.ResourceRecord {
	for i, rec := range originalRecords {
		if r.isOwnedByThisClass(rec) {
			return append(originalRecords[:i], originalRecords[i+1:]...)
		}
	}
	return originalRecords
}
