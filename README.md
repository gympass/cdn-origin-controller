[![Codacy Badge](https://app.codacy.com/project/badge/Grade/fc9d3d6690714fe79af21149955633c2)](https://www.codacy.com/gh/Gympass/cdn-origin-controller/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Gympass/cdn-origin-controller&amp;utm_campaign=Badge_Grade)
[![Codacy Badge](https://app.codacy.com/project/badge/Coverage/fc9d3d6690714fe79af21149955633c2)](https://www.codacy.com/gh/Gympass/cdn-origin-controller/dashboard?utm_source=github.com&utm_medium=referral&utm_content=Gympass/cdn-origin-controller&utm_campaign=Badge_Coverage)

# cdn-origin-controller

cdn-origin-controller is a Kubernetes controller to attach CDN origins based on Ingress resources. This is made possible by configuring your Ingress resources with certain annotations, which tell the controller how these origins should be created.

Currently, the controller only supports adding origins to AWS CloudFront. Other CDN providers may become supported based on community use cases.

Requirements:

  - Kubernetes with Ingresses on networking.k8s.io/v1beta1 (< v1.22)

# AWS CloudFront

The controller will look for three locations within the Ingress definition in order to determine how the origin and behaviors should be created:

  - `Ingress.status.loadbalancer.ingress[].host`: domains of the origins will be retrieved from here.
  - `Ingress.spec.rules[].http.paths[].path`: for each path at least one behavior will be created, allowing different cache behavior for different backends, for example.
  - `Ingress.spec.rules[].http.paths[].pathType`: in order to determine how to create each behavior while replicating routing that is expected from each path type. For `ImplementationSpecific` the value is simply copied as the behavior's path pattern.

The following annotation controls how origins and behaviors are attached to existing CloudFront distributions:

  - `cdn-origin-controller.gympass.com/cdn.id`: the ID of the CloudFront distribution where the origins and behaviors should be present. Example: `cdn-origin-controller.gympass.com/cdn.id: E7IQHB92RC62FG`

The controller needs permission to manipulate the CloudFront distributions. A [sample IAM Policy](docs/iam_policy.json) is provided with the necessary IAM actions.

> **Important**: This sample policy grants the necessary actions for proper functioning of the controller, but it grants them on all CloudFront distributions. Changing this policy to make it more restrictive and secure is encouraged.

# Configuration

Use the following environment variables to change the controller's behavior:

| Env var key | Description                                                                                                                                   | Default |
|-------------|-----------------------------------------------------------------------------------------------------------------------------------------------|---------|
| LOG_LEVEL   | Represents log level of verbosity. Can be "debug", "info", "warn", "error", "dpanic", "panic" and "fatal" (sorted with decreasing verbosity). | info    |
| DEV_MODE    | When set to "true" logs in unstructured text instead of JSON. Also overrides LOG_LEVEL to "debug".                                            | false   |

# Contributing

Please open an issue in order to report bugs, ask questions or discuss the controller.

If you would like to contribute with code, please refer to our [Contributor's Guide](CONTRIBUTING.md).
