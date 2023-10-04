# cdn-origin-controller

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/ca2a2f38c1be40e5b4d94b25ad2134fd)](https://www.codacy.com/gh/Gympass/cdn-origin-controller/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Gympass/cdn-origin-controller&amp;utm_campaign=Badge_Grade)
[![Codacy Badge](https://app.codacy.com/project/badge/Coverage/ca2a2f38c1be40e5b4d94b25ad2134fd)](https://www.codacy.com/gh/Gympass/cdn-origin-controller/dashboard?utm_source=github.com&utm_medium=referral&utm_content=Gympass/cdn-origin-controller&utm_campaign=Badge_Coverage)

cdn-origin-controller is a Kubernetes controller to provision CDNs based on Ingress resources. This is made possible by configuring your Ingress resources with certain annotations, which tell the controller how origins should be configured at the CDN.

The controller allows infrastructure engineers to provide infrastructure configuration of the CDN via environment variables while allowing developers to configure each origin through Ingresses, maintaining a clean cut between infrastructure and application contexts.

Currently, the controller only supports adding origins to AWS CloudFront. Other CDN providers may become supported based on community use cases.

Requirements:

- Kubernetes with Ingress support for networking.k8s.io/v1

## AWS CloudFront

The controller will look for three locations within the Ingress definition in order to determine how the origin and behaviors should be created:

- `Ingress.status.loadbalancer.ingress[].host`: domains of the origins will be retrieved from here.
- `Ingress.spec.rules[].http.paths[].path`: for each path at least one behavior will be created, allowing different cache behavior for different backends, for example.
- `Ingress.spec.rules[].http.paths[].pathType`: in order to determine how to create each behavior while replicating routing that is expected from each path type. For `ImplementationSpecific` the value is simply copied as the behavior's path pattern.

The following annotation controls how origins and behaviors are attached to CloudFront distributions:

- `cdn-origin-controller.gympass.com/cdn.group`: a CDN group should be used to bind Ingress resources together under the same distribution. Required. If the group does not exist yet a new distribution will be provisioned. Example: `cdn-origin-controller.gympass.com/cdn.group: customer-portal`
- `cdn-origin-controller.gympass.com/cdn.class`: the [CDN class](#cdn-classes) this Ingress resource belongs to. Required. Must match the CDN Class configured for the controller deployment that is meant to manage this Ingress.
- `cdn-origin-controller.gympass.com/cf.alternate-domain-names`: a comma-separated list of alternate domains to be configured on the CloudFront distribution. Duplicates on the same or different Ingress resources from the same group cause no harm. Example: `alias1.foo,alias2.foo`
- `cdn-origin-controller.gympass.com/cf.origin-request-policy`: the ID of the origin request policy that should be associated with the behaviors defined by the Ingress resource. Defaults to the ID of the AWS pre-defined policy "Managed-AllViewer" (ID: 216adef6-5c7f-47e4-b989-5492eafa07d3) for Public origins, and "Managed-CORS-S3Origin" (ID: 88a5eaf4-2fd4-4709-b370-b4c650ea3fcf) for Bucket origins, however these defaults can be overriden through configuration by setting the `CF_DEFAULT_PUBLIC_ORIGIN_ACCESS_REQUEST_POLICY_ID` or `CF_DEFAULT_PUBLIC_ORIGIN_ACCESS_REQUEST_POLICY_ID` environment variables. If set to`"None"` no policy will be associated.
- `cdn-origin-controller.gympass.com/cf.cache-policy`: the ID of the cache policy that should be associated with the behaviors defined by the Ingress resource. Defaults to the ID of the AWS pre-defined policy "CachingDisabled" (ID: 4135ea2d-6df8-44a3-9df3-4b5a84be39ad), this default can be overriden by setting the `CF_DEFAULT_CACHE_REQUEST_POLICY_ID` environment variable. More details about managed cache policies [see](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-cache-policies.html).
- `cdn-origin-controller.gympass.com/cf.origin-response-timeout`: the number of seconds that CloudFront waits for a response from the origin, from 1 to 60. Example: `"30"`
- `cdn-origin-controller.gympass.com/cf.viewer-function-arn`: the ARN of the CloudFront function you would like to associate to viewer requests in each behavior managed by this Ingress. Example: `arn:aws:cloudfront::000000000000:function/my-function`
- `cdn-origin-controller.gympass.com/cf.web-acl-arn`: A unique identifier that specifies the AWS WAF web ACL, if any, to associate with this distribution. To specify a web ACL created using the latest version of AWS WAF, use the ACL ARN, for example `arn:aws:wafv2:us-east-1:123456789012:global/webacl/ExampleWebACL/473e64fd-f30b-4765-81a0-62ad96dd167a`. To specify a web ACL created using AWS WAF Classic, use the ACL ID, for example `473e64fd-f30b-4765-81a0-62ad96dd167a`.
- `cdn-origin-controller.gympass.com/cf.tags`: A map of key/value strings to be configured in Cloudfront distribution. The value of this annotation should be given as a YAML map. Example:

  ```yaml
  cdn-origin-controller.gympass.com/cf.tags: |
    mykey1: myvalue1
    mykey2: myvalue2
  ```

The controller needs permission to manipulate the CloudFront distributions. A [sample IAM Policy](docs/iam_policy.json) is provided with the necessary IAM actions.

> **Important**: This sample policy grants the necessary actions for proper functioning of the controller, but it grants them on all CloudFront distributions. Changing this policy to make it more restrictive and secure is encouraged.

## CDN Classes

The controller has several [infrastructure configurations](#configuration). In order to support different controller configurations running in the same cluster it's possible to make each of them responsible for a class. This is done using the `CDNClass` Kubernetes kind.

### Parameters

| Parameter      | Required | Description                                                                                                                                                      |   |   |
|----------------|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------|---|---|
| certificateArn | yes      | The ARN of ACM certificate which should be used by the distributions.                                                                                            |   |   |
| hostedZoneID   | yes      | The ID of the Route53 zone where the aliases should be created in.                                                                                               |   |   |
| createAlias    | yes      | Whether the controller should create DNS records for a distribution's alternate domain names.                                                                    |   |   |
| txtOwnerValue  | yes      | The controller creates TXT records for managing aliases. In it, a value is written to bind that given record to a particular instance of the controller running. |   |   |

For example, imagine you need some of your CloudFront distributions to be in the `foo.com` zone and the others on the `bar.com` zone. In order to do that you need create both `CDNClass` kinds and set different values for the `hostedZoneID`, `certificateArn`, `createAlias` and `txtOwnerValue` parameters.

For this example, for the first kind we should have:

```yaml
apiVersion: cdn.gympass.com/v1alpha1
kind: CDNClass
metadata:
  name: foo-com
spec:
  certificateArn: "<Certificate ARN from given hosted zone>"
  hostedZoneID: "<foo-com hosted zone ID>"
  createAlias: true
  txtOwnerValue: "<foo-owner value>"
```

While the other kind is defined with:

```yaml
apiVersion: cdn.gympass.com/v1alpha1
kind: CDNClass
metadata:
  name: bar-com
spec:
  certificateArn: "<Certificate ARN from given hosted zone>"
  hostedZoneID: "<bar-com hosted zone ID>"
  createAlias: true
  txtOwnerValue: "<bar-owner value>"
```

In order for Ingresses to be part of one class or the other they must have cdn class annotation set the respective value.

Ingresses that serve as origins for the CloudFront at the `foo.com` zone should have the following annotation:

``` yaml
cdn-origin-controller.gympass.com/cdn.class: foo-com
```

While Ingresses that serve as origins for CloudFronts at the `bar.com` zone should have:

``` yaml
cdn-origin-controller.gympass.com/cdn.class: bar-com
```

## Behavior ordering

During reconciliation, the controller will assemble desired behaviors based on all
Ingresses that compose a single CloudFront. In order to determine the correct
order of behaviors based on their paths, the following criteria are followed.

Given a path `i` and a path `j`:

1. `i` is more specific if `i` is longer than `j`

2. If both are the same length, `i` is more specific if it comes before `j` in a
 special alphabetical order, where "*" and "?" come after all other characters.

The order can be summarized as `[0-9],[A-Z],[a-z], ?, *`.

"?" is considered more specific than "\*" because it represents a single char,
while "\*" may represent more.

This allows for more specific routing of requests which could be matched by better
routes.

For example:

- Catch-all Ingress, with a path `/*/foo`
- USA-specific Ingress, with a path `/en-us/foo`

In CloudFront, these would result in the following order:

- `/en-us/foo` -> en-us specific origin
- `/*/foo` -> catch all origin

## User-supplied origin/behavior configuration

If you need additional origin/behavior configuration that you can't express via Ingress resources (e.g., pointing to an S3 bucket with static resources of your application) you can do that using the `cdn-origin-controller.gympass.com/cf.user-origins`.

The value for this annotation is a YAML list, with each item representing a single origin with all cache behavior-related configuration associated to that origin. For example:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: foobar
  annotations:
    cdn-origin-controller.gympass.com/cdn.group: "foobar"
    cdn-origin-controller.gympass.com/cf.user-origins: |
      - host: foo.com
        originAccess: Bucket
        responseTimeout: 30
        paths:
          - /foo
      - host: bar.com
        originAccess: Public
        originRequestPolicy: None
        viewerFunctionARN: "bar:arn"
        webACLARN: "arn:aws:wafv2:us-east-1:123456789012:global/webacl/ExampleWebACL/473e64fd-f30b-4765-81a0-62ad96dd167a"
        paths:
          - /bar
          - /bar/*
```

The `.host` is the hostname of the origin you're configuring.

The `.paths` field is a list of strings representing the cache behavior paths that should be configured.

The `.originAccess` field allows for different origin access configurations:

- Public, the default value if the field is omitted, should be used when the origin is publicly accessible, such as an Amazon S3 bucket that is configured with static website hosting;
- Bucket should be used if the origin is an S3 bucket that is not configured with static website hosting, see the [additional configuration section](#bucket-origin-access);

Each remaining field has a corresponding annotation value, [documented in a dedicated section](#aws-cloudfront).

The table below maps remaining available fields of an entry in this list to an annotation:

| Entry field          | Annotation                                                   |
|----------------------|--------------------------------------------------------------|
| .originRequestPolicy | cdn-origin-controller.gympass.com/cf.origin-request-policy   |
| .responseTimeout     | cdn-origin-controller.gympass.com/cf.origin-response-timeout |
| .viewerFunctionARN   | cdn-origin-controller.gympass.com/cf.viewer-function-arn     |
| .cachePolicy         | cdn-origin-controller.gympass.com/cf.cache-policy            |
| .webACLARN           | cdn-origin-controller.gympass.com/cf.web-acl-arn             |

### Bucket origin access

When `.originAccess` is set to `Bucket`, the `.host` should be the Bucket endpoint following the pattern \<bucket-name>.s3.\<region>.amazonaws.com. Check the [AWS documentation on using S3 as a CloudFront origin](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/DownloadDistS3AndCustomOrigins.html#using-s3-as-origin).

When using S3 as an origin, all requests sent from CloudFront to the S3 bucket will be authenticated. Origin Access Control (OAC) will be configured automatically by this controller, but the user will have to configure a policy, in the S3 bucket, to allow access to CloudFront. Please check the [AWS documentation on the required bucket policy](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-restricting-access-to-s3.html#create-oac-overview-s3).

NOTE: When using Origin Access Control, CloudFront will always override the client's authorization header, in order to be able to authenticate with S3. Make sure your specific S3 bucket doesn't have any additional custom authentication layer, which could break CloudFront access.

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
| CF_AWS_WAF                 | No       | The Web ACL which should be associated with the distributions. Use the ID for WAF v1 and the ARN for WAF v2.                                                                                                                                                                                                                                                 | ""                                    |
| CF_CUSTOM_TAGS             | No       | Comma-separated list of custom tags to be added to distributions. Example: "foo=bar,bar=foo"                                                                                                                                                                                                                                                                 | ""                                    |
| CF_DEFAULT_ORIGIN_DOMAIN   | Yes      | Domain of the default origin each distribution must have to route traffic to in case no custom behaviors match the request.                                                                                                                                                                                                                                  | ""                                    |
| CF_DESCRIPTION_TEMPLATE    | No       | Template of the distribution's description. Currently a single field can be accessed, `{{group}}`, which matches the CDN group under which the distribution was provisioned.                                                                                                                                                                                 | "Serve contents for {{group}} group." |
| CF_ENABLE_IPV6             | No       | Whether the distribution should also expose an IPv6 address to serve requests.                                                                                                                                                                                                                                                                               | "true"                                |
| CF_ENABLE_LOGGING          | No       | If set to true enables sending logs to CloudWatch; `CF_S3_BUCKET_LOG` must be set as well.                                                                                                                                                                                                                                                                   | "false"                               |
| CF_PRICE_CLASS             | Yes      | The distribution price class. Possible values are: "PriceClass_All", "PriceClass_200", "PriceClass_100". [Official reference](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/PriceClass.html).                                                                                                                                           | "PriceClass_All"                      |
| CF_S3_BUCKET_LOG           | No       | The domain of the S3 bucket CloudWatch logs should be sent to. Each distribution will have its own directory inside the bucket with the same as the distribution's group. For example, if the group is "foo", the logs will be stored as `foo/<ID>.<timestamp and hash>.gz`.<br><br> If `CF_ENABLE_LOGGING` is not set to "true" then this value is ignored. | ""                                    |
| CF_SECURITY_POLICY         | No       | The TLS/SSL security policy to be used when serving requests. [Official reference](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/secure-connections-supported-viewer-protocols-ciphers.html). <br><br> Must also inform a valid `CF_CUSTOM_SSL_CERT` if set.                                                                            | ""                                    |
| DEV_MODE                   | No       | When set to "true" logs in unstructured text instead of JSON. Also overrides LOG_LEVEL to "debug".                                                                                                                                                                                                                                                           | "false"                               |
| LOG_LEVEL                  | No       | Represents log level of verbosity. Can be "debug", "info", "warn", "error", "dpanic", "panic" and "fatal" (sorted with decreasing verbosity).                                                                                                                                                                                                                | "info"                                |
| ENABLE_DELETION            | No       | Represent whether CloudFront Distributions and Route53 records should be deleted based on Ingresses being deleted. Ownership TXT DNS records are also not deleted to allow for self-healing in case of accidental deletion of Kubernetes resources.                                                                                                          | "false"                               |

## Contributing

Please open an issue in order to report bugs, ask questions or discuss the controller.

If you would like to contribute with code, please refer to our [Contributor's Guide](CONTRIBUTING.md).
