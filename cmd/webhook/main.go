package main

import (
	"context"
	"os"

	"github.com/QuanZhang-William/pod-affinity/pkg/mutatingwebhook/pipelinerun_mutatingwebhook"
	"github.com/QuanZhang-William/pod-affinity/pkg/mutatingwebhook/pod_mutatingwebhook"
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

func newPodMutatingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return pod_mutatingwebhook.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"pod.mutation.webhook.pod-affinity.tekton.dev",

		// The path on which to serve the webhook.
		"/pod-mutating",

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return ctx
		},
	)
}

func newPipelineRunMutatingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return pipelinerun_mutatingwebhook.NewPipelineRunAdmissionController(ctx,

		// Name of the resource webhook.
		"pipelinerun.mutation.webhook.pod-affinity.tekton.dev",

		// The path on which to serve the webhook.
		"/pr-mutating",

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
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

	// Set up a signal context with our webhook options
	ctx := webhook.WithOptions(signals.NewContext(), webhook.Options{
		ServiceName: serviceName,
		Port:        8443,
		SecretName:  secretName,
	})

	sharedmain.MainWithContext(ctx, "tekton-pod-affinity-webhook",
		certificates.NewController,
		newPodMutatingAdmissionController,
		newPipelineRunMutatingAdmissionController,
	)
}
