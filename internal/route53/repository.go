package route53

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

const (
	upsertAction           = "UPSERT"
	cfHostedZoneId         = "Z2FDTNDATAQYW2"
	cfEvaluateTargetHealth = false
	hostTypeA              = "A"
	hostTypeAAAA           = "AAAA"
)

type AliasRepository interface {
}

type repository struct {
	awsClient    route53iface.Route53API
	hostedZoneId string
}

func NewRoute53AliasRepository(awsClient route53iface.Route53API, hostedZoneId string) AliasRepository {
	return &repository{awsClient: awsClient, hostedZoneId: hostedZoneId}
}

func (r repository) Sync(aliases Aliases) error {
	var changes []*route53.Change

	for _, e := range aliases.Entries {
		if aliases.Ipv6Enabled {
			changes = append(changes, r.newChange(e, aliases.Target, hostTypeAAAA))
		}
		changes = append(changes, r.newChange(e, aliases.Target, hostTypeA))
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
			Comment: aws.String("Cloudfront distribution managed by operator"),
		},
		HostedZoneId: aws.String(r.hostedZoneId),
	}

	if _, err := r.awsClient.ChangeResourceRecordSets(input); err != nil {
		return fmt.Errorf("syncing alias: %v", err)
	}

	return nil
}

func (r repository) newChange(entry, target, hostType string) *route53.Change {
	return &route53.Change{
		Action: aws.String(upsertAction),
		ResourceRecordSet: &route53.ResourceRecordSet{
			AliasTarget: &route53.AliasTarget{
				DNSName:              aws.String(target),
				EvaluateTargetHealth: aws.Bool(cfEvaluateTargetHealth),
				HostedZoneId:         aws.String(cfHostedZoneId),
			},
			Name: aws.String(entry),
			Type: aws.String(hostType),
		},
	}
}
