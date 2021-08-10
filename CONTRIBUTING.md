# Contributing

When contributing to this repository, please first discuss the change you wish to make via [issue](https://github.com/Gympass/cdn-origin-controller/issues),
email, or any other method with the owners of this repository before making a change.

Please note we have a [code of conduct](https://github.com/Gympass/cdn-origin-controller/blob/main/CODE_OF_CONDUCT.md), please follow it in all your interactions with the project.

## Making a Pull Request

1. Fork and clone this repo
2. Create your local branch
2. Make sure that you have all [requirements](#requirements-to-run-locally) to run the project locally
3. Always [run tests](#running-tests) before sending a PR to make sure the license headers and the manifests are updated (and of course the unit tests are passing)
4. Submit a pull request against the upstream source repository

### Requirements to run locally

* Go 1.16
* Operator SDK 1.10
* Local Kubernetes 1.19 e.g.: [minikube](https://minikube.sigs.k8s.io/), [k3d](https://k3d.io/), [kind](https://kind.sigs.k8s.io/)

### Running tests
To run tests locally, run the following command:

```sh
make test
```

### Running application
To run application, run the following command:
```sh
make run
```
