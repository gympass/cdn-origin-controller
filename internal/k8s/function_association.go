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

package k8s

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FunctionType string

const (
	FunctionTypeEdge       FunctionType = "edge"
	FunctionTypeCloudfront FunctionType = "cloudfront"
)

const (
	cfFunctionAssociationsAnnotation = "cdn-origin-controller.gympass.com/cf.function-associations"
)

type FunctionAssociationsPaths struct {
	Paths map[string]FunctionAssociations `yaml:",inline"`
}

type FunctionAssociations struct {
	ViewerRequest  *ViewerRequestFunction `yaml:"viewerRequest"`
	ViewerResponse *ViewerFunction        `yaml:"viewerResponse"`
	OriginRequest  *OriginRequestFunction `yaml:"originRequest"`
	OriginResponse *OriginFunction        `yaml:"originResponse"`
}

func (fa FunctionAssociations) Validate() error {
	if err := fa.ViewerRequest.validate(); err != nil {
		return fmt.Errorf("invalid viewerRequest: %v", err)
	}
	if err := fa.ViewerResponse.validate(); err != nil {
		return fmt.Errorf("invalid viewerResponse: %v", err)
	}
	if err := fa.OriginRequest.validate(); err != nil {
		return fmt.Errorf("invalid originRequest: %v", err)
	}
	if err := fa.OriginResponse.validate(); err != nil {
		return fmt.Errorf("invalid originResponse: %v", err)
	}

	if !fa.ViewerRequest.isCompatibleWith(fa.ViewerResponse) {
		return fmt.Errorf(
			"different function types informed for viewerResponse and viewerRequest. They must be %q only or %q only",
			FunctionTypeCloudfront, FunctionTypeEdge)
	}

	return nil
}

func (fa FunctionAssociations) Merge(other FunctionAssociations) (FunctionAssociations, error) {
	merged := fa.deepCopy()
	other = other.deepCopy()

	if fa.ViewerRequest != nil && other.ViewerRequest != nil {
		if fa.ViewerRequest.ARN != other.ViewerRequest.ARN {
			return FunctionAssociations{}, fmt.Errorf("viewer request function informed twice with different ARNs")
		}
		if fa.ViewerRequest.FunctionType != other.ViewerRequest.FunctionType {
			return FunctionAssociations{}, fmt.Errorf("viewer request function informed twice with different function types")
		}
		if fa.ViewerRequest.IncludeBody != other.ViewerRequest.IncludeBody {
			return FunctionAssociations{}, fmt.Errorf("viewer request function informed twice with different body inclusion configuration")
		}
	} else if other.ViewerRequest != nil {
		merged.ViewerRequest = other.ViewerRequest
	}

	if fa.ViewerResponse != nil && other.ViewerResponse != nil {
		if fa.ViewerResponse.ARN != other.ViewerResponse.ARN {
			return FunctionAssociations{}, fmt.Errorf("viewer response function informed twice with different ARNs")
		}
		if fa.ViewerResponse.FunctionType != other.ViewerResponse.FunctionType {
			return FunctionAssociations{}, fmt.Errorf("viewer response function informed twice with different function types")
		}
	} else if other.ViewerResponse != nil {
		merged.ViewerResponse = other.ViewerResponse
	}

	if fa.OriginRequest != nil && other.OriginRequest != nil {
		if fa.OriginRequest.ARN != other.OriginRequest.ARN {
			return FunctionAssociations{}, fmt.Errorf("origin request function informed twice with different ARNs")
		}
		if fa.OriginRequest.IncludeBody != other.OriginRequest.IncludeBody {
			return FunctionAssociations{}, fmt.Errorf("origin request function informed twice with different body inclusion configuration")
		}
	} else if other.OriginRequest != nil {
		merged.OriginRequest = other.OriginRequest
	}

	if fa.OriginResponse != nil && other.OriginResponse != nil {
		if fa.OriginResponse.ARN != other.OriginResponse.ARN {
			return FunctionAssociations{}, fmt.Errorf("origin response function informed twice with different ARNs")
		}
	} else if other.OriginResponse != nil {
		merged.OriginResponse = other.OriginResponse
	}

	return merged, nil
}

func (fa FunctionAssociations) deepCopy() FunctionAssociations {
	copied := FunctionAssociations{}
	if fa.ViewerRequest != nil {
		copied.ViewerRequest = &ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          fa.ViewerRequest.ARN,
				FunctionType: fa.ViewerRequest.FunctionType,
			},
			IncludeBody: fa.ViewerRequest.IncludeBody,
		}
	}
	if fa.ViewerResponse != nil {
		copied.ViewerResponse = &ViewerFunction{
			ARN:          fa.ViewerResponse.ARN,
			FunctionType: fa.ViewerResponse.FunctionType,
		}
	}

	if fa.OriginRequest != nil {
		copied.OriginRequest = &OriginRequestFunction{
			OriginFunction: OriginFunction{ARN: fa.OriginRequest.ARN},
			IncludeBody:    fa.OriginRequest.IncludeBody,
		}
	}
	if fa.OriginResponse != nil {
		copied.OriginResponse = &OriginFunction{ARN: fa.OriginResponse.ARN}
	}

	return copied
}

type ViewerFunction struct {
	ARN          string       `yaml:"arn"`
	FunctionType FunctionType `yaml:"functionType"`
}

func (f *ViewerFunction) validate() error {
	if f == nil {
		return nil
	}

	if len(f.ARN) == 0 {
		return errors.New("function arn must be informed")
	}

	if f.FunctionType != FunctionTypeEdge && f.FunctionType != FunctionTypeCloudfront {
		return fmt.Errorf("invalid function type: %q", f.FunctionType)
	}

	return nil
}

type ViewerRequestFunction struct {
	ViewerFunction `yaml:",inline"`
	IncludeBody    bool `yaml:"includeBody"`
}

func (f *ViewerRequestFunction) validate() error {
	if f == nil {
		return nil
	}

	if err := f.ViewerFunction.validate(); err != nil {
		return err
	}

	if f.ViewerFunction.FunctionType == FunctionTypeCloudfront && f.IncludeBody {
		return fmt.Errorf("includeBody is only supported for functionType %q", FunctionTypeEdge)
	}

	return nil
}

func (f *ViewerRequestFunction) isCompatibleWith(response *ViewerFunction) bool {
	if f == nil || response == nil {
		return true
	}
	return f.FunctionType == response.FunctionType
}

type OriginFunction struct {
	ARN string `yaml:"arn"`
}

func (f *OriginFunction) validate() error {
	if f == nil {
		return nil
	}

	if len(f.ARN) == 0 {
		return errors.New("function arn must be informed")
	}

	return nil
}

type OriginRequestFunction struct {
	OriginFunction `yaml:",inline"`
	IncludeBody    bool `yaml:"includeBody"`
}

func (f *OriginRequestFunction) validate() error {
	if f == nil {
		return nil
	}

	if err := f.OriginFunction.validate(); err != nil {
		return err
	}

	return nil
}

func functionAssociations(obj client.Object) (map[string]FunctionAssociations, error) {
	faYAML, ok := obj.GetAnnotations()[cfFunctionAssociationsAnnotation]
	if !ok {
		return nil, nil
	}

	fa := FunctionAssociationsPaths{}
	err := yaml.Unmarshal([]byte(faYAML), &fa)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling YAML: %v", err)
	}

	return fa.Paths, nil
}
