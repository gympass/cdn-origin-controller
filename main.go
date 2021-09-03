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

	"github.com/aws/aws-sdk-go/aws/session"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"github.com/joho/godotenv"
	"go.uber.org/zap/zapcore"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	//+kubebuilder:scaffold:imports
	"github.com/Gympass/cdn-origin-controller/controllers"
	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
	"github.com/Gympass/cdn-origin-controller/internal/config"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
	_ = godotenv.Load()
}

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

	operatorCfg := config.Parse()

	ctrl.SetLogger(zap.New(
		zap.UseFlagOptions(&opts),
		zap.UseDevMode(operatorCfg.DevMode),
		zap.Level(mustGetLogLevel(operatorCfg.LogLevel)),
	))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "19e16908.gympass.com",
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

	ingressReconciler := controllers.IngressReconciler{
		Recorder: mgr.GetEventRecorderFor("cdn-origin-controller"),
		Repo:     cloudfront.NewOriginRepository(mustGetCloudFrontClient()),
	}

	v1beta1Reconciler := controllers.V1beta1Reconciler{
		Client:            mgr.GetClient(),
		OriginalLog:       ctrl.Log.WithName("controllers").WithName("ingressv1beta1"),
		Scheme:            mgr.GetScheme(),
		IngressReconciler: ingressReconciler,
	}

	if err := v1beta1Reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up reconciler")
		os.Exit(1)
	}

	v1Reconciler := controllers.V1Reconciler{
		Client:            mgr.GetClient(),
		OriginalLog:       ctrl.Log.WithName("controllers").WithName("ingressv1"),
		Scheme:            mgr.GetScheme(),
		IngressReconciler: ingressReconciler,
	}

	if err := v1Reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up reconciler")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
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

func mustGetCloudFrontClient() cloudfrontiface.CloudFrontAPI {
	s := session.Must(session.NewSession())
	return awscloudfront.New(s)
}
