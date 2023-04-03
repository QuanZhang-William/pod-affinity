# PipelineRun Pod Affinity

This prototype works as a "plugin" to the Tekton Pipeline project. This prototype ensures that All the pods created by a PipelineRun are scheduled 
to the same node.

## Installation
[Install Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/main/docs/install.md)
and [disable the affinity assistant](https://github.com/tektoncd/pipeline/blob/main/docs/additional-configs.md#customizing-the-pipelines-controller-behavior).

Build and install from source with [ko](https://ko.build/):

```sh
ko apply -f config
```

## How it works
In this prototype, the "one worker per PipelineRun" feature is implemented by [K8s Pod Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity). 
This prototype contains 3 parts:

### PipelineRun Webhook
A separate PipelineRun mutation webhook is used to mark a PipelineRun in 'Pending' status when created.

### PipelineRun Controller
A separate PipelineRun controller is used to create a placeholder StatefulSet based on the PipelineRun name, 
and triggers the PipelineRun by cancelling the `pending` status (so that th PipelineRun controller in Tekton Pipeline project
can then start to reconcile the PipelineRun normally).

The PipelineRun controller also cleans up the placeholder StatefulSet when a PipelineRun is completed.

### Pod Webhook
A separate Pod mutatin webhook is used to add Pod affinity terms (based on PipelineRun name) to all the pods Created by a PipelineRun, 
so that all the pods are anchored to the placeholder SS to acheive "one worker per PipelineRun" feature