package podaffinity

import (
	"context"

	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	logging "knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler implements controller.Reconciler
type Reconciler struct {
	PipelineClientSet clientset.Interface
	PipelineRunLister listers.PipelineRunLister
}

func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	logger.Infof("Quan Test: The pipeline run reconciler triggered, namespace: %v, name: %v", pr.Namespace, pr.Name)
	return nil
}
