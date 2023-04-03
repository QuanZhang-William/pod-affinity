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
	mergeAffinityWithAffinityAssistant(p.Spec.Affinity, podAffinityName)
}

func mergeAffinityWithAffinityAssistant(affinity *corev1.Affinity, podAffinityName string) {
	podAffinityTerm := podAffinityTermUsingAffinityAssistant(podAffinityName)

	if affinity.PodAffinity == nil {
		affinity.PodAffinity = &corev1.PodAffinity{}
	}

	affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution =
		append(affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, *podAffinityTerm)
}

func podAffinityTermUsingAffinityAssistant(affinityAssistantName string) *corev1.PodAffinityTerm {
	return &corev1.PodAffinityTerm{LabelSelector: &metav1.LabelSelector{
		MatchLabels: map[string]string{
			workspace.LabelInstance:  affinityAssistantName,
			workspace.LabelComponent: workspace.ComponentNameAffinityAssistant,
		},
	},
		TopologyKey: "kubernetes.io/hostname",
	}
}
