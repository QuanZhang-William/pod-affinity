package podaffinity

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	"github.com/tektoncd/pipeline/pkg/reconciler/volumeclaim"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

const (
	ControllerName = "custom-pod-affinity-controller"
)

func NewController(opts *pipeline.Options) func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		kubeclientset := kubeclient.Get(ctx)
		pipelineClientSet := pipelineclient.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)

		/*configStore := config.NewStore(logger.Named("config-store"))
		configStore.WatchConfigs(cmw)*/
		r := &Reconciler{
			PipelineRunLister: pipelineRunInformer.Lister(),
			PipelineClientSet: pipelineClientSet,
			KubeClientSet:     kubeclientset,
			pvcHandler:        volumeclaim.NewPVCHandler(kubeclientset, logger),
			Images:            opts.Images,
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
