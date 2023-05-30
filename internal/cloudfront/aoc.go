package cloudfront

import awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"

type AOC struct {
	ID                            string `json:"id"`
	Name                          string `json:"name"`
	OriginName                    string `json:"originName"`
	OriginAccessControlOriginType string `json:"originAccessControlOriginType"`
	SigningBehavior               string `json:"signingBehavior"`
	SigningProtocol               string `json:"signingProtocol"`
}

func NewAOC(originName string) AOC {
	return AOC{
		Name:                          originName,
		OriginName:                    originName,
		OriginAccessControlOriginType: awscloudfront.OriginAccessControlOriginTypesS3,
		SigningBehavior:               awscloudfront.OriginAccessControlSigningBehaviorsAlways,
		SigningProtocol:               awscloudfront.OriginAccessControlSigningProtocolsSigv4,
	}
}
