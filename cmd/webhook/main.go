package main

import (
	"context"
	"os"

	"github.com/QuanZhang-William/pod-affinity/pkg/api/mutatingwebhook"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/certificates"
)

const (
	// WebhookLogKey is the name of the logger for the webhook cmd.
	// This name is also used to form lease names for the leader election of the webhook's controllers.
	WebhookLogKey = "tekton-pod-affinity-webhook"
)

func newMutatingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	//Quan TODO: check if watch config is needed
	/*
		store := defaultconfig.NewStore(logging.FromContext(ctx).Named("config-store"))
		store.WatchConfigs(cmw)
	*/
	return mutatingwebhook.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"mutation.webhook.pod-affinity.tekton.dev",

		// The path on which to serve the webhook.
		"/mutating",

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		// Quan TODO: check if context func is fine
		func(ctx context.Context) context.Context {
			return ctx
		},
	)
}

func main() {
	serviceName := os.Getenv("WEBHOOK_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "tekton-pod-affinity-webhook"
	}

	secretName := os.Getenv("WEBHOOK_SECRET_NAME")
	if secretName == "" {
		secretName = "tekton-pod-affinity-webhook-certs"
	}

	// Scope informers to the webhook's namespace instead of cluster-wide
	// Quan: see if PRs created in default ns will trigger it or not
	//ctx := injection.WithNamespaceScope(signals.NewContext(), "tekton-pod-affinity")

	// Set up a signal context with our webhook options
	ctx := webhook.WithOptions(signals.NewContext(), webhook.Options{
		ServiceName: serviceName,
		Port:        8443,
		SecretName:  secretName,
	})

	sharedmain.MainWithContext(ctx, "tekton-pod-affinity-webhook",
		certificates.NewController,
		newMutatingAdmissionController,
	)
}
