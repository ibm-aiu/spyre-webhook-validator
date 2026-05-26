/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package main

import (
	"os"

	"github.com/ibm-aiu/spyre-webhook-validator/pkg/validator"
	uberzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	clusterPolicyValidator = validator.NewClusterPolicyHandler()
	podHookHandler         = validator.AdmissionHandler{Validator: validator.NewPodValidator(clusterPolicyValidator)}
	jobHookHandler         = validator.AdmissionHandler{Validator: validator.NewJobValidator(clusterPolicyValidator)}
	deploymentHookHandler  = validator.AdmissionHandler{Validator: validator.NewDeploymentValidator(clusterPolicyValidator)} //nolint:lll
	clusterPolicyHandler   = validator.AdmissionHandler{Validator: clusterPolicyValidator}
	loglevel               = zapcore.InfoLevel
)

func prepareLogger() {
	// enable zap logger to generate controller-runtime log entries
	if logLevelConfig, exists := os.LookupEnv(("LOGLEVEL")); exists {
		if l, err := zapcore.ParseLevel(logLevelConfig); err == nil {
			loglevel = l
		}
	}
	levelEnabler := uberzap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= loglevel
	})
	opts := zap.Options{
		Development: true,
		Level:       levelEnabler,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}

	zlogger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(zlogger)
	klog.SetLogger(zlogger)
	ctrl.Log.Info("controller-runtime log is set", "loglevel", loglevel.String())
	klog.Infof("validator log is set with loglevel: %s", loglevel.String())
}

func main() {
	prepareLogger()

	webhookCertPath := "/etc/webhook/certs"
	webhookPort := 8443
	probePort := ":8081"

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		HealthProbeBindAddress: probePort,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookPort,
			CertDir: webhookCertPath,
		}),
	})
	if err != nil {
		ctrl.Log.Error(err, "Could not create a new manager")
		os.Exit(1)
	}
	hookServer := mgr.GetWebhookServer()
	hookServer.Register("/validate-pods", &webhook.Admission{Handler: podHookHandler})
	hookServer.Register("/validate-jobs", &webhook.Admission{Handler: jobHookHandler})
	hookServer.Register("/validate-deployments", &webhook.Admission{Handler: deploymentHookHandler})
	hookServer.Register("/validate-clusterpolicy", &webhook.Admission{Handler: clusterPolicyHandler})
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		ctrl.Log.Error(err, "Could not add health check to manager")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", hookServer.StartedChecker()); err != nil {
		ctrl.Log.Error(err, "Could not add readiness check to manager")
		os.Exit(1)
	}
	ctrl.Log.Info("Starting webhook")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Errorf("Problem running webhook server: %v", err)
		os.Exit(1)
	}
}
