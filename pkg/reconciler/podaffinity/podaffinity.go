package podaffinity

import (
	"context"
	"fmt"

	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/reconciler/volumeclaim"
	"k8s.io/client-go/kubernetes"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logging "knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler implements controller.Reconciler
type Reconciler struct {
	PipelineClientSet clientset.Interface
	PipelineRunLister listers.PipelineRunLister
	KubeClientSet     kubernetes.Interface
	pvcHandler        volumeclaim.PvcHandler
	Images            pipeline.Images
}

func customPodAffinityRequired(pr *v1beta1.PipelineRun) bool {
	// if this PR is not intended for pod affinity experiment, ignore
	if _, found := pr.Labels["tekton.dev/custom-pod-affinity"]; !found {
		return false
	}

	// if not in pending status or the pending status is not marked by the custom webhook, ignore
	if _, found := pr.Labels["tekton.dev/marked-pending-by-webhook"]; !found || pr.Spec.Status != v1beta1.PipelineRunSpecStatusPending {
		return false
	}

	return true
}

// ReconcileKind removes the tekton.dev/marked-pending-by-webhook label and cancel the pending status of PR
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	logger.Infof(" The pipeline run reconciler triggered, namespace: %v, name: %v", pr.Namespace, pr.Name)

	name := getPodAffinityValue(pr.Name)
	if pr.IsDone() {
		// clean placeholder statefulset
		if err := r.KubeClientSet.AppsV1().StatefulSets(pr.Namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			logger.Errorf("Clean up placeholder SS failed: %v", err)
			return err
		}

		logger.Infof("Statefulset delete properly")
		return nil
	}

	if !customPodAffinityRequired(pr) {
		logger.Infof("Reconcile skipped for pr: %v, status: %v", pr.Name, pr.Spec.Status)
		return nil
	}

	// for now we assume we don't use persistent volume in PR
	// we must create pvcs from templates before the placeholder pod to achieve "volume scheduling"
	/*if pr.HasVolumeClaimTemplate() {
		// create workspace PVC from template
		if err := r.pvcHandler.CreatePersistentVolumeClaimsForWorkspaces(ctx, pr.Spec.Workspaces, *kmeta.NewControllerRef(pr), pr.Namespace); err != nil {
			logger.Errorf("Failed to create PVC for PipelineRun %s: %v", pr.Name, err)
			pr.Status.MarkFailed(volumeclaim.ReasonCouldntCreateWorkspacePVC,
				"Failed to create PVC for PipelineRun %s/%s Workspaces correctly: %s",
				pr.Namespace, pr.Name, err)
			return controller.NewPermanentError(err)
		}
	}*/

	// create a placeholder pod
	_, err := r.KubeClientSet.AppsV1().StatefulSets(pr.Namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		podAffinitySS := SimplePlaceholderStatefulSet(pr, r.Images.NopImage, false)
		_, err := r.KubeClientSet.AppsV1().StatefulSets(pr.Namespace).Create(ctx, podAffinitySS, metav1.CreateOptions{})
		if err != nil {
			logger.Fatalf("Failed to create StateulSet: %v", err)
		}
		if err == nil {
			logger.Infof("Created StatefulSet %s in namespace %s", name, pr.Namespace)
		}
	case err != nil:
		logger.Fatalf("Failed to get StateulSet: %v", err)
	}

	// cancel the pending status of the pipelinerun
	/*
		failed to update pipeline run: Operation cannot be fulfilled on pipelineruns.tekton.dev \"demo-set-name6hmb5\":
		 the object has been modified; please apply your changes to the latest version and try again

		 but it still works...
	*/
	newPR, err := r.PipelineRunLister.PipelineRuns(pr.Namespace).Get(pr.Name)
	if err != nil {
		return fmt.Errorf("error getting PipelineRun %s in namespace %s when updating labels: %w", pr.Name, pr.Namespace, err)
	}
	newPR = newPR.DeepCopy()
	// Properly merge labels and annotations, as the labels *might* have changed during the reconciliation
	newPR.Labels = pr.Labels

	if _, found := pr.Labels["tekton.dev/marked-pending-by-webhook"]; found {
		delete(newPR.Labels, "tekton.dev/marked-pending-by-webhook")
		newPR.Spec.Status = ""
	}
	_, err = r.PipelineClientSet.TektonV1beta1().PipelineRuns(pr.Namespace).Update(ctx, newPR, metav1.UpdateOptions{})
	logger.Infof("marked-pending-by-webhook canceled and pending status canceled")
	return err
}
