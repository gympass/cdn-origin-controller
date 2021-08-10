# cdn-origin-controller

cdn-origin-controller is a Kubernetes controller to attach CDN origins based on Ingress resources. This is made possible by configuring your Ingress resources with certain annotations, which tell the controller how these origins should be created.

Currently, the controller only supports adding origins to AWS CloudFront. Other CDN providers may become supported based on community use cases.

# Contributing

Please open an issue in order to report bugs, ask questions or discuss the controller.

If you would like to contribute with code, please refer to our [Contributor's Guide](CONTRIBUTING.md).