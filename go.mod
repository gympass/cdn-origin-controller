module github.com/Gympass/cdn-origin-controller

go 1.16

require (
	github.com/aws/aws-sdk-go v1.40.21
	github.com/go-logr/logr v0.4.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/joho/godotenv v1.3.0
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.17.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)
