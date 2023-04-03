package podaffinity

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/workspace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MutatePodAffinity applies the pod affinity terms based on the PipelineRun name
func MutatePodAffinity(ctx context.Context, p *corev1.Pod, pipelineRunName string) {
	// for now we assume the original pod has no pod affinity
	if p.Spec.Affinity == nil {
		p.Spec.Affinity = &corev1.Affinity{}
	}

	podAffinityName := getPodAffinityValue(pipelineRunName)
	podAffinityTerm := podAffinityTermUsingPlaceholderPod(podAffinityName)

	affinity := p.Spec.Affinity
	if affinity.PodAffinity == nil {
		affinity.PodAffinity = &corev1.PodAffinity{}
	}
	affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution =
		append(affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, *podAffinityTerm)
}

func podAffinityTermUsingPlaceholderPod(placeholderPodName string) *corev1.PodAffinityTerm {
	return &corev1.PodAffinityTerm{LabelSelector: &metav1.LabelSelector{
		MatchLabels: map[string]string{
			workspace.LabelInstance:  placeholderPodName,
			workspace.LabelComponent: ComponentNamePlaceholder,
		},
	},
		TopologyKey: "kubernetes.io/hostname",
	}
}
