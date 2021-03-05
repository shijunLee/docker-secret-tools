package main

import (
	"context"
	"github.com/shijunLee/docker-secret-tools/pkg/config"
	"github.com/shijunLee/docker-secret-tools/pkg/controller"
	"github.com/shijunLee/docker-secret-tools/pkg/log"
	"github.com/shijunLee/docker-secret-tools/pkg/utils"
	"github.com/shijunLee/docker-secret-tools/pkg/webhook"
	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	cfgFile := ""
	logLevel := ""
	logFile := ""
	port := 0
	pflag.StringVarP(&cfgFile, "config", "c", "", "set the default config file dir")
	pflag.StringVarP(&logLevel, "logLevel", "", "error", "tools log level")
	pflag.StringVarP(&logFile, "logFile", "", "/log/tools.log", "tools log")
	pflag.IntVarP(&port, "port", "", 0, "server start port")
	pflag.Parse()
	config.InitConfig(cfgFile)
	// init log
	logOptions := &zap.Options{}
	level := zapcore.ErrorLevel
	err := level.UnmarshalText([]byte(logLevel))
	if err == nil {
		logOptions.Level = level
	} else {
		logOptions.Level = zapcore.ErrorLevel
	}
	log.InitLog(logOptions, logFile)
	ctrl.SetLogger(log.Logger)
	setupLog := ctrl.Log.WithName("setup")
	runtimeScheme := runtime.NewScheme()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  runtimeScheme,
		MetricsBindAddress:      "0",
		Port:                    9443,
		LeaderElection:          true,
		LeaderElectionID:        "7982b436.tools.domain",
		LeaderElectionNamespace: utils.GetCurrentNameSpace(),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if err = (&controller.NamespaceReconciler{
		Client:            mgr.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("NamespaceReconciler"),
		DockerSecretNames: config.GlobalConfig.DockerSecretNames,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NamespaceReconciler")
		os.Exit(1)
	}
	if port > 0 {
		config.GlobalConfig.ServerPort = port
	}

	switch config.GlobalConfig.SetMethod {
	case config.SetMethodWebHook:
		server := webhook.NewServer(mgr, config.GlobalConfig.DockerSecretNames, config.GlobalConfig.ServerPort)
		server.Start(context.Background())
	case config.SetMethodUpdate:
		if err = (&controller.WorkloadReconciler{
			Client:            mgr.GetClient(),
			Log:               ctrl.Log.WithName("controllers").WithName("NamespaceReconciler"),
			DockerSecretNames: config.GlobalConfig.DockerSecretNames,
			Object:            &corev1.Pod{},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "NamespaceReconciler")
			os.Exit(1)
		}
		if err = (&controller.WorkloadReconciler{
			Client:            mgr.GetClient(),
			Log:               ctrl.Log.WithName("controllers").WithName("NamespaceReconciler"),
			DockerSecretNames: config.GlobalConfig.DockerSecretNames,
			Object:            &appsv1.Deployment{},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "NamespaceReconciler")
			os.Exit(1)
		}
		if err = (&controller.WorkloadReconciler{
			Client:            mgr.GetClient(),
			Log:               ctrl.Log.WithName("controllers").WithName("NamespaceReconciler"),
			DockerSecretNames: config.GlobalConfig.DockerSecretNames,
			Object:            &appsv1.StatefulSet{},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "NamespaceReconciler")
			os.Exit(1)
		}
		if err = (&controller.WorkloadReconciler{
			Client:            mgr.GetClient(),
			Log:               ctrl.Log.WithName("controllers").WithName("NamespaceReconciler"),
			DockerSecretNames: config.GlobalConfig.DockerSecretNames,
			Object:            &appsv1.ReplicaSet{},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "NamespaceReconciler")
			os.Exit(1)
		}
		if err = (&controller.WorkloadReconciler{
			Client:            mgr.GetClient(),
			Log:               ctrl.Log.WithName("controllers").WithName("NamespaceReconciler"),
			DockerSecretNames: config.GlobalConfig.DockerSecretNames,
			Object:            &appsv1.DaemonSet{},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "NamespaceReconciler")
			os.Exit(1)
		}
	}
	stopSignalHandler := ctrl.SetupSignalHandler()
	setupLog.Info("starting manager")
	if err := mgr.Start(stopSignalHandler); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
