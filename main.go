// Copyright (c) 2021 GPBR Participacoes LTDA.
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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	awsroute53 "github.com/aws/aws-sdk-go/service/route53"
	"github.com/joho/godotenv"
	"go.uber.org/zap/zapcore"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	cdnv1alpha1 "github.com/Gympass/cdn-origin-controller/api/v1alpha1"
	"github.com/Gympass/cdn-origin-controller/internal/k8s"

	//+kubebuilder:scaffold:imports
	"github.com/Gympass/cdn-origin-controller/controllers"
	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/route53"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(cdnv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
	_ = godotenv.Load()
}

const (
	leaderElectionID = "cdn-origin.gympass.com"
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	var opts zap.Options
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	cfg := config.Parse()

	ctrl.SetLogger(zap.New(
		zap.UseFlagOptions(&opts),
		zap.UseDevMode(cfg.DevMode),
		zap.Level(mustGetLogLevel(cfg.LogLevel)),
	))
	setupLog.V(1).Info("Config parsed.", "config", cfg)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	mustSetupControllers(mgr, cfg)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func mustSetupControllers(mgr manager.Manager, cfg config.Config) {
	s := session.Must(session.NewSession())

	cfClient := awscloudfront.New(s)

	distRepo := cloudfront.DistRepository{
		CloudFrontClient: cfClient,
		OACRepo:          cloudfront.NewOACRepository(cfClient, cloudfront.NewOACLister(cfClient), cfg),
		TaggingClient:    resourcegroupstaggingapi.New(s),
		CallerRef:        func() string { return time.Now().String() },
		WaitTimeout:      time.Minute * 10,
		Cfg:              cfg,
	}
	distRepo.RunPostCreationOperations = distRepo.Sync

	cfService := &cloudfront.Service{
		Client:    mgr.GetClient(),
		Recorder:  mgr.GetEventRecorderFor("cdn-origin-controller"),
		DistRepo:  distRepo,
		AliasRepo: route53.NewAliasRepository(awsroute53.New(s), cfg),
		Config:    cfg,
	}

	const ingressVersionAvailableMsg = " Ingress available, setting up its controller. Other versions will not be tried."

	setupLog.V(1).Info(networkingv1.SchemeGroupVersion.String() + ingressVersionAvailableMsg)
	cdnClassFetcher := k8s.NewCDNClassFetcher(mgr.GetClient())
	cfService.Fetcher = k8s.NewIngressFetcherV1(mgr.GetClient(), cdnClassFetcher)
	mustSetupV1Controller(mgr, cfService, cdnClassFetcher)
}

func mustSetupV1Controller(mgr manager.Manager, ir *cloudfront.Service, cdnClassFetcher k8s.CDNClassFetcher) {
	v1Reconciler := controllers.V1Reconciler{
		Client:            mgr.GetClient(),
		CloudFrontService: ir,
		CDNClassFetcher:   cdnClassFetcher,
	}

	if err := v1Reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up v1 ingress controller")
		os.Exit(1)
	}
}

func mustGetLogLevel(logLvl string) zapcore.Level {
	var l zapcore.Level
	if err := l.Set(logLvl); err != nil {
		panic(fmt.Errorf("invalid log level config: %v", err))
	}
	return l
}
