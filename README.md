# cdn-origin-controller

cdn-origin-controller is a Kubernetes controller to attach CDN origins based on Ingress resources. This is made possible by configuring your Ingress resources with certain annotations, which tell the controller how these origins should be created.

Currently, the controller only supports adding origins to AWS CloudFront. Other CDN providers may become supported based on community use cases.

Requirements:

  - Kubernetes v1.19 or higher

# AWS CloudFront

The controller will look for three locations within the Ingress definition in order to determine how the origin and behaviors should be created:

  - `Ingress.status.loadbalancer.ingress[].<host/ip>`: domains of the origins will be retrieved from here.
  - `Ingress.spec.rules[].http.paths[].path`: a behavior for each path will be created, allowing different cache behavior for different backends, for example.
  - `Ingress.spec.rules[].http.paths[].pathType`: in order to determine whether to use wildcards or not. For `Prefix` an "*" is appended to the path when defining the behavior.

The following annotation controls how origins and behaviors are attached to existing CloudFront distributions:

  - `cdn-origin-controller.gympass.com/cdn.id`: the ID of the CloudFront distribution where the origins and behaviors should be present. Example: `cdn-origin-controller.gympass.com/cdn.id: E7IQHB92RC62FG` 

# Configuration

Use the following environment variables to change the controller's behavior:

| Env var key | Description                                                                                                                                   | Default |
|-------------|-----------------------------------------------------------------------------------------------------------------------------------------------|---------|
| LOG_LEVEL   | Represents log level of verbosity. Can be "debug", "info", "warn", "error", "dpanic", "panic" and "fatal" (sorted with decreasing verbosity). | info    |
| DEV_MODE    | When set to "true" logs in unstructured text instead of JSON. Also overrides LOG_LEVEL to "debug".                                            | false   |

# Contributing

Please open an issue in order to report bugs, ask questions or discuss the controller.

If you would like to contribute with code, please refer to our [Contributor's Guide](CONTRIBUTING.md).