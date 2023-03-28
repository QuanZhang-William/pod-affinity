package podaffinity

import (
	"context"

	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

const (
	ControllerName = "custom-pod-affinity-controller"
)

func NewController() func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		pipelineClientSet := pipelineclient.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)

		/*configStore := config.NewStore(logger.Named("config-store"))
		configStore.WatchConfigs(cmw)*/
		r := &Reconciler{
			PipelineRunLister: pipelineRunInformer.Lister(),
			PipelineClientSet: pipelineClientSet,
		}
		impl := pipelinerunreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName:         ControllerName,
				SkipStatusUpdates: true, // Don't update PipelineRun status. This is the responsibility of Tekton Pipelines
				//ConfigStore:       configStore,
			}
		})

		logger.Info("Setting up event handlers")
		pipelineRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
		return impl
	}
}
