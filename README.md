# cdn-origin-controller

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/ca2a2f38c1be40e5b4d94b25ad2134fd)](https://www.codacy.com/gh/Gympass/cdn-origin-controller/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Gympass/cdn-origin-controller&amp;utm_campaign=Badge_Grade)
[![Codacy Badge](https://app.codacy.com/project/badge/Coverage/ca2a2f38c1be40e5b4d94b25ad2134fd)](https://www.codacy.com/gh/Gympass/cdn-origin-controller/dashboard?utm_source=github.com&utm_medium=referral&utm_content=Gympass/cdn-origin-controller&utm_campaign=Badge_Coverage)

cdn-origin-controller is a Kubernetes controller to provision CDNs based on Ingress resources. This is made possible by configuring your Ingress resources with certain annotations, which tell the controller how origins should be configured at the CDN.

The controller allows infrastructure engineers to provide infrastructure configuration of the CDN via environment variables while allowing developers to configure each origin through Ingresses, maintaining a clean cut between infrastructure and application contexts.

Currently, the controller only supports adding origins to AWS CloudFront. Other CDN providers may become supported based on community use cases.

Requirements:

- Kubernetes with Ingress support for networking.k8s.io/v1 or networking.k8s.io/v1beta1

## AWS CloudFront

The controller will look for three locations within the Ingress definition in order to determine how the origin and behaviors should be created:

- `Ingress.status.loadbalancer.ingress[].host`: domains of the origins will be retrieved from here.
- `Ingress.spec.rules[].http.paths[].path`: for each path at least one behavior will be created, allowing different cache behavior for different backends, for example.
- `Ingress.spec.rules[].http.paths[].pathType`: in order to determine how to create each behavior while replicating routing that is expected from each path type. For `ImplementationSpecific` the value is simply copied as the behavior's path pattern.

The following annotation controls how origins and behaviors are attached to CloudFront distributions:

- `cdn-origin-controller.gympass.com/cdn.group`: a CDN group should be used to bind Ingress resources together under the same distribution. Required. If the group does not exist yet a new distribution will be provisioned. Example: `cdn-origin-controller.gympass.com/cdn.group: customer-portal`
- `cdn-origin-controller.gympass.com/cdn.class`: the [CDN class](#cdn-classes) this Ingress resource belongs to. Required. Must match the CDN Class configured for the controller deployment that is meant to manage this Ingress.
- `cdn-origin-controller.gympass.com/cf.alternate-domain-names`: a comma-separated list of alternate domains to be configured on the CloudFront distribution. Duplicates on the same or different Ingress resources from the same group cause no harm. Example: `alias1.foo,alias2.foo`
- `cdn-origin-controller.gympass.com/cf.origin-request-policy`: the ID of the origin request policy that should be associated with the behaviors defined by the Ingress resource. Defaults to the ID of the AWS pre-defined policy "Managed-AllViewer" (ID: 216adef6-5c7f-47e4-b989-5492eafa07d3). If set to `"None"` no policy will be associated.  
- `cdn-origin-controller.gympass.com/cf.cache-policy`: the ID of the cache policy that should be associated with the behaviors defined by the Ingress resource. Defaults to the ID of the AWS pre-defined policy "CachingDisabled" (ID: 4135ea2d-6df8-44a3-9df3-4b5a84be39ad). More details about managed cache policies [see](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-cache-policies.html).
- `cdn-origin-controller.gympass.com/cf.origin-response-timeout`: the number of seconds that CloudFront waits for a response from the origin, from 1 to 60. Example: `"30"`
- `cdn-origin-controller.gympass.com/cf.viewer-function-arn`: the ARN of the CloudFront function you would like to associate to viewer requests in each behavior managed by this Ingress. Example: `arn:aws:cloudfront::000000000000:function/my-function`
- `cdn-origin-controller.gympass.com/cf.web-acl-arn`: A unique identifier that specifies the AWS WAF web ACL, if any, to associate with this distribution. To specify a web ACL created using the latest version of AWS WAF, use the ACL ARN, for example `arn:aws:wafv2:us-east-1:123456789012:global/webacl/ExampleWebACL/473e64fd-f30b-4765-81a0-62ad96dd167a`. To specify a web ACL created using AWS WAF Classic, use the ACL ID, for example `473e64fd-f30b-4765-81a0-62ad96dd167a`.

The controller needs permission to manipulate the CloudFront distributions. A [sample IAM Policy](docs/iam_policy.json) is provided with the necessary IAM actions.

> **Important**: This sample policy grants the necessary actions for proper functioning of the controller, but it grants them on all CloudFront distributions. Changing this policy to make it more restrictive and secure is encouraged.

## CDN Classes

The controller has several [infrastructure configurations](#configuration). In order to support different controller configurations running in the same cluster it's possible to make each of them responsible for a class. This is done using the `CDN_CLASS` environment variable.

For example, imagine you need some of your CloudFront distributions to be in the `foo.com` zone and the others on the `bar.com` zone. In order to do that you need to set different values for the `CF_ROUTE53_HOSTED_ZONE_ID` variable. Additionally, you need each deployment to have a unique CDN class, so you can tell them apart. 

For this example, let's say the first deployment will have:
```
CF_ROUTE53_HOSTED_ZONE_ID=<ID of the foo.com zone>
CDN_CLASS="foo-com"
```

While the other deployment is defined with:
```
CF_ROUTE53_HOSTED_ZONE_ID=<ID of the bar.com zone>
CDN_CLASS="bar-com"
```

In order for Ingresses to be part of one class or the other they must have cdn class annotation set the respective value.

Ingresses that serve as origins for the CloudFront at the `foo.com` zone should have the following annotation:

```
cdn-origin-controller.gympass.com/cdn.class: foo-com
```

While Ingresses that serve as origins for CloudFronts at the `bar.com` zone should have:

```
cdn-origin-controller.gympass.com/cdn.class: bar-com
```

## User-supplied origin/behavior configuration

If you need additional origin/behavior configuration that you can't express via Ingress resources (e.g., pointing to an S3 bucket with static resources of your application) you can do that using the `cdn-origin-controller.gympass.com/cf.user-origins`.

The value for this annotation is a YAML list, with each item representing a single origin with all cache behavior-related configuration associated to that origin. For example:

```
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: foobar
  annotations:
    cdn-origin-controller.gympass.com/cdn.group: "foobar"
    cdn-origin-controller.gympass.com/cf.user-origins: |
      - host: foo.com
        responseTimeout: 30
        paths:
          - /foo
      - host: bar.com
        originRequestPolicy: None
        viewerFunctionARN: "bar:arn"
        webACLARN: "arn:aws:wafv2:us-east-1:123456789012:global/webacl/ExampleWebACL/473e64fd-f30b-4765-81a0-62ad96dd167a"
        paths:
          - /bar
          - /bar/*
```

The `.host` is the hostname of the origin you're configuring. The `.paths` field is a list of strings representing the cache behavior paths that should be configured. Each remaining field has a corresponding annotation value, [documented in a dedicated section](#aws-cloudfront).

The table below maps remaining available fields of an entry in this list to an annotation:

| Entry field          | Annotation                                                   |
|----------------------|--------------------------------------------------------------|
| .originRequestPolicy | cdn-origin-controller.gympass.com/cf.origin-request-policy   |
| .responseTimeout     | cdn-origin-controller.gympass.com/cf.origin-response-timeout |
| .viewerFunctionARN   | cdn-origin-controller.gympass.com/cf.viewer-function-arn     |
| .cachePolicy         | cdn-origin-controller.gympass.com/cf.cache-policy            |
| .webACLARN           | cdn-origin-controller.gympass.com/cf.web-acl-arn             |

## CDNStatus custom resource

The controller provides a [custom Kubernetes resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) for providing user feedback on a managed CDN. It's a cluster-scoped resource, meaning it's unique across the entire cluster and is part of no namespace.

The controller automatically creates these resources based on the Ingresses it reconciles. The name of the resource is equal to the Ingresses `cdn-origin-controller.gympass.com/cdn.group` annotation, meaning there is a 1:1 mapping between CDNStatus resources and provisioned CDNs. 

For example, if there was a group of Ingresses with this annotation set to `"foo"` and another group of Ingresses set to `"bar"`, you would end up with two CDNStatus resources. To list them:
```bash
$ kubectl get cdnstatus 
NAME    ID               ALIASES                       ADDRESS
foo     A4NX3S1AJ7ZGH7   ["alias1.com","alias2.com"]   k7zxergbqey2lg.cloudfront.net
bar     BH0C38HF34OFT6   ["alias3.com","alias4.com"]   kg3gwck75ewn98.cloudfront.net
```

You can also get more information by describing a particular resource, including which Ingresses compose the desired configuration of that particular distribution:

```bash
$ kubectl describe cdnstatus foo
Name:         foo
Namespace:    
Labels:       <none>
Annotations:  <none>
API Version:  cdn.gympass.com/v1alpha1
Kind:         CDNStatus
Metadata:
  Creation Timestamp:  2021-11-05T18:32:02Z
  Generation:          1
  Resource Version:    847526546
  UID:                 8d8141f7-bbc8-44dc-b9b4-8a2ad73582ab
Status:
  Address:  k7zxergbqey2lg.cloudfront.net
  Aliases:
    alias1.com
    alias2.com
  Arn:  arn:aws:cloudfront::000000000000:distribution/A4NX3S1AJ7ZGH7
  Id:   A4NX3S1AJ7ZGH7
  Ingresses:
    default/app1: Synced
    default/app2: Synced
    default/app3: Failed
Events:
  Type    Reason                  Age   From                      Message
  ----    ------                  ----  ----                      -------
  Normal  SuccessfullyReconciled  20s   cdn-origin-controller     default/app1: Successfully reconciled CDN
  Normal  SuccessfullyReconciled  19s   cdn-origin-controller     default/app2: Successfully reconciled CDN
  Warning FailedToReconcile       12s   cdn-origin-controller     default/app3: Unable to reconcile CDN: some error
  
```

The events are also replicated to the specific Ingress resources which were being reconciled.

> **Important**: the controller relies on this resource to maintain state of which Ingresses are part of a distribution. It's recommended to configure RBAC to only allow the controller and cluster administrators to perform writes against this resource.

## Installing via Helm

Access the [documentation](https://gympass.github.io/cdn-origin-controller/) to install the cdn-origin-controller using a helm chart repository.

## Configuration

Use the following environment variables to change the controller's behavior:

| Env var key                | Required | Description                                                                                                                                                                                                                                                                                                                                                  | Default                               |
|----------------------------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|
| CDN_CLASS                  | No       | The class identifier used to determine if a given Ingress should be managed by this controller instance. See the [dedicated section](#cdn-classes) for more details.                                                                                                                                                                                         | "default"                             |
| CF_AWS_WAF                 | No       | The Web ACL which should be associated with the distributions. Use the ID for WAF v1 and the ARN for WAF v2.                                                                                                                                                                                                                                                 | ""                                    |
| CF_CUSTOM_SSL_CERT         | No       | The ARN of ACM certificate which should be used by the distributions. <br><br> Must also inform a valid `CF_SECURITY_POLICY` if set.                                                                                                                                                                                                                         | ""                                    |
| CF_CUSTOM_TAGS             | No       | Comma-separated list of custom tags to be added to distributions. Example: "foo=bar,bar=foo"                                                                                                                                                                                                                                                                 | ""                                    |
| CF_DEFAULT_ORIGIN_DOMAIN   | Yes      | Domain of the default origin each distribution must have to route traffic to in case no custom behaviors match the request.                                                                                                                                                                                                                                  | ""                                    |
| CF_DESCRIPTION_TEMPLATE    | No       | Template of the distribution's description. Currently a single field can be accessed, `{{group}}`, which matches the CDN group under which the distribution was provisioned.                                                                                                                                                                                 | "Serve contents for {{group}} group." |
| CF_ENABLE_IPV6             | No       | Whether the distribution should also expose an IPv6 address to serve requests.                                                                                                                                                                                                                                                                               | "true"                                |
| CF_ENABLE_LOGGING          | No       | If set to true enables sending logs to CloudWatch; `CF_S3_BUCKET_LOG` must be set as well.                                                                                                                                                                                                                                                                   | "false"                               |
| CF_PRICE_CLASS             | Yes      | The distribution price class. Possible values are: "PriceClass_All", "PriceClass_200", "PriceClass_100". [Official reference](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/PriceClass.html).                                                                                                                                           | "PriceClass_All"                      |
| CF_ROUTE53_CREATE_ALIAS    | No       | Whether the controller should create DNS records for a distribution's alternate domain names. If IPv6 is enabled (see `CF_ENABLE_IPV6`) AAAA and A records are created. Only A records are created otherwise. <br><br> Must also set `CF_ROUTE53_HOSTED_ZONE_ID` and `CF_ROUTE53_TXT_OWNER_VALUE` if set to "true".                                          | "false"                               |
| CF_ROUTE53_HOSTED_ZONE_ID  | No       | The ID of the Route53 zone where the aliases should be created in.                                                                                                                                                                                                                                                                                           | ""                                    |
| CF_ROUTE53_TXT_OWNER_VALUE | No       | The controller creates TXT records for managing aliases. In it, a value written to bind that given record to a particular instance of the controller running. Use a unique value for each instance. Example: "coc-staging-ab64sj2".                                                                                                                          | ""                                    |
| CF_S3_BUCKET_LOG           | No       | The domain of the S3 bucket CloudWatch logs should be sent to. Each distribution will have its own directory inside the bucket with the same as the distribution's group. For example, if the group is "foo", the logs will be stored as `foo/<ID>.<timestamp and hash>.gz`.<br><br> If `CF_ENABLE_LOGGING` is not set to "true" then this value is ignored. | ""                                    |
| CF_SECURITY_POLICY         | No       | The TLS/SSL security policy to be used when serving requests. [Official reference](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/secure-connections-supported-viewer-protocols-ciphers.html). <br><br> Must also inform a valid `CF_CUSTOM_SSL_CERT` if set.                                                                            | ""                                    |
| DEV_MODE                   | No       | When set to "true" logs in unstructured text instead of JSON. Also overrides LOG_LEVEL to "debug".                                                                                                                                                                                                                                                           | "false"                               |
| LOG_LEVEL                  | No       | Represents log level of verbosity. Can be "debug", "info", "warn", "error", "dpanic", "panic" and "fatal" (sorted with decreasing verbosity).                                                                                                                                                                                                                | "info"                                |
| ENABLE_DELETION            | No       | Represent whether CloudFront Distributions and Route53 records should be deleted based on Ingresses being deleted. Ownership TXT DNS records are also not deleted to allow for self-healing in case of accidental deletion of Kubernetes resources.                                                                                                          | "false"                               |

## Contributing

Please open an issue in order to report bugs, ask questions or discuss the controller.

If you would like to contribute with code, please refer to our [Contributor's Guide](CONTRIBUTING.md).
