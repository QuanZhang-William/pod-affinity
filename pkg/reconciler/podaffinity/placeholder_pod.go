package podaffinity

import (
	"crypto/sha256"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/workspace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"
)

// SimplePlaceholderStatefulSet has similar functionality to PlaceholderStatefulSet
// with no pod template or volume is applied to the placeholder SS
// TODO: make use useAntiPodAffinity configurable
func SimplePlaceholderStatefulSet(pr *v1beta1.PipelineRun, placeholderPodImage string, useAntiPodAffinity bool) *appsv1.StatefulSet {
	// We want a singleton pod
	replicas := int32(1)

	// We do not use this for now
	tpl := &pod.AffinityAssistantTemplate{}

	containers := []corev1.Container{{
		Name:  "affinity-assistant",
		Image: placeholderPodImage,
		Args:  []string{"tekton_run_indefinitely"},

		// Set requests == limits to get QoS class _Guaranteed_.
		// See https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed
		// Affinity Assistant pod is a placeholder; request minimal resources
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("50m"),
				"memory": resource.MustParse("100Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("50m"),
				"memory": resource.MustParse("100Mi"),
			},
		},
	}}

	name := getPodAffinityValue(pr.Name)
	placeholderSS := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          getStatefulSetLabels(pr, name),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(pr)},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getStatefulSetLabels(pr, name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getStatefulSetLabels(pr, name),
				},
				Spec: corev1.PodSpec{
					Containers: containers,

					Tolerations:      tpl.Tolerations,
					NodeSelector:     tpl.NodeSelector,
					ImagePullSecrets: tpl.ImagePullSecrets,
				},
			},
		},
	}

	if useAntiPodAffinity {
		placeholderSS.Spec.Template.Spec.Affinity = getPlaceholderMergedWithPodTemplateRequiredAffinity(pr)
	}

	return placeholderSS
}

// PlaceholderStatefulSet is not currently being used as we need to figure out PV availability zone concern
// TODO: check if we apply pipeline pod template to placeholder pod
func PlaceholderStatefulSet(name string, pr *v1beta1.PipelineRun, claimName string, placeholderPodImage string, defaultAATpl *pod.AffinityAssistantTemplate, useAntiPodAffinity bool) *appsv1.StatefulSet {
	// We want a singleton pod
	replicas := int32(1)

	tpl := &pod.AffinityAssistantTemplate{}
	// merge pod template from spec and default if any of them are defined

	if pr.Spec.PodTemplate != nil || defaultAATpl != nil {
		tpl = pod.MergeAAPodTemplateWithDefault(pr.Spec.PodTemplate.ToAffinityAssistantTemplate(), defaultAATpl)
	}

	containers := []corev1.Container{{
		Name:  "affinity-assistant",
		Image: placeholderPodImage,
		Args:  []string{"tekton_run_indefinitely"},

		// Set requests == limits to get QoS class _Guaranteed_.
		// See https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed
		// Affinity Assistant pod is a placeholder; request minimal resources
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("50m"),
				"memory": resource.MustParse("100Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("50m"),
				"memory": resource.MustParse("100Mi"),
			},
		},
	}}

	placeholderSS := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          getStatefulSetLabels(pr, name),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(pr)},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getStatefulSetLabels(pr, name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getStatefulSetLabels(pr, name),
				},
				Spec: corev1.PodSpec{
					Containers: containers,

					Tolerations:      tpl.Tolerations,
					NodeSelector:     tpl.NodeSelector,
					ImagePullSecrets: tpl.ImagePullSecrets,

					Affinity: getPlaceholderMergedWithPodTemplateRequiredAffinity(pr),
					Volumes: []corev1.Volume{{
						Name: "workspace",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								// A Pod mounting a PersistentVolumeClaim that has a StorageClass with
								// volumeBindingMode: Immediate
								// the PV is allocated on a Node first, and then the pod need to be
								// scheduled to that node.
								// To support those PVCs, the Affinity Assistant must also mount the
								// same PersistentVolumeClaim - to be sure that the Affinity Assistant
								// pod is scheduled to the same Availability Zone as the PV, when using
								// a regional cluster. This is called VolumeScheduling.
								ClaimName: claimName,
							}},
					}},
				},
			},
		},
	}

	if useAntiPodAffinity {
		placeholderSS.Spec.Template.Spec.Affinity = getPlaceholderMergedWithPodTemplateRequiredAffinity(pr)
	}

	return placeholderSS
}

func getStatefulSetLabels(pr *v1beta1.PipelineRun, placeholderPodName string) map[string]string {
	// Propagate labels from PipelineRun to StatefulSet.
	labels := make(map[string]string, len(pr.ObjectMeta.Labels)+1)
	for key, val := range pr.ObjectMeta.Labels {
		labels[key] = val
	}
	labels[pipeline.PipelineRunLabelKey] = pr.Name

	// LabelInstance is used to configure PodAffinity for all TaskRuns belonging to this PipelineRun
	// LabelComponent is used to configure PodAntiAffinity to other PipelineRun
	labels[LabelInstance] = placeholderPodName
	labels[LabelComponent] = ComponentNamePlaceholder
	return labels
}

// getPlaceholderMergedWithPodTemplateRequiredAffinity return the affinity that merged with PipelineRun PodTemplate affinity (required).
func getPlaceholderMergedWithPodTemplateRequiredAffinity(pr *v1beta1.PipelineRun) *corev1.Affinity {
	// use podAntiAffinity to repel other affinity assistants
	repelOtherPlaceholderPodAffinityTerm := corev1.PodAffinityTerm{LabelSelector: &metav1.LabelSelector{
		MatchLabels: map[string]string{
			workspace.LabelComponent: ComponentNamePlaceholder,
		},
	},
		TopologyKey: "kubernetes.io/hostname",
	}

	PlaceholderAffinity := &corev1.Affinity{}
	if pr.Spec.PodTemplate != nil && pr.Spec.PodTemplate.Affinity != nil {
		PlaceholderAffinity = pr.Spec.PodTemplate.Affinity
	}
	if PlaceholderAffinity.PodAntiAffinity == nil {
		PlaceholderAffinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}

	// we use RequiredDuringSchedulingIgnoredDuringExecution to enforce repeling other placeholder pods here;
	// we use PreferedDuringSchedulingIgnoredDuringExecution in Tekton Pipelines
	PlaceholderAffinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution =
		append(PlaceholderAffinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			repelOtherPlaceholderPodAffinityTerm)

	return PlaceholderAffinity
}

func getPodAffinityValue(pipelineRunName string) string {
	hashBytes := sha256.Sum256([]byte(pipelineRunName))
	hashString := fmt.Sprintf("%x", hashBytes)
	return fmt.Sprintf("%s-%s", "custom-pod-affinity", hashString[:10])
}
