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

- `cdn-origin-controller.gympass.com/cdn.group`: a CDN group should be used to bind Ingress resources together under the same distribution. If the group does not exist yet a new distribution will be provisioned. Example: `cdn-origin-controller.gympass.com/cdn.group: customer-portal`
- `cdn-origin-controller.gympass.com/cf.viewer-function-arn`: the ARN of the CloudFront function you would like to associate to viewer requests in each behavior managed by this Ingress. Example: `arn:aws:cloudfront::000000000000:function/my-function`
- `cdn-origin-controller.gympass.com/cf.origin-response-timeout`: the number of seconds that CloudFront waits for a response from the origin, from 1 to 60. Example: `30`

The controller needs permission to manipulate the CloudFront distributions. A [sample IAM Policy](docs/iam_policy.json) is provided with the necessary IAM actions.

> **Important**: This sample policy grants the necessary actions for proper functioning of the controller, but it grants them on all CloudFront distributions. Changing this policy to make it more restrictive and secure is encouraged.

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

## Contributing

Please open an issue in order to report bugs, ask questions or discuss the controller.

If you would like to contribute with code, please refer to our [Contributor's Guide](CONTRIBUTING.md).
