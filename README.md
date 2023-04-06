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

Set a StorageClass with `WaitForFirstConsumer` VolumeBindingMode [as default](https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/) to avoid availability zone scheduling conflict by running: 

```sh
 kubectl patch storageclass custom -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

Please see more details about [PV availabity zone](https://kubernetes.io/docs/reference/labels-annotations-taints/#topologykubernetesiozone) and
[StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/)


To use this feature, please add a label `tekton.dev/custom-pod-affinity: "true"` to your PipelineRun.

## How it works
In this prototype, the "one Node per PipelineRun" feature is implemented by [K8s Pod Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity). 
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
so that all the pods are anchored to the placeholder SS to acheive "one Node per PipelineRun" feature

## Examples
List out all the nodes:

```bash
$ kubectl get nodes
NAME                                                  STATUS   ROLES    AGE    VERSION
gke-tekton-pipeline-test-default-pool-07b93c60-2xbi   Ready    <none>   33d    v1.24.8-gke.2000
gke-tekton-pipeline-test-default-pool-55a03fc3-45dl   Ready    <none>   140m   v1.24.8-gke.2000
gke-tekton-pipeline-test-default-pool-55a03fc3-t4jh   Ready    <none>   121m   v1.24.8-gke.2000
gke-tekton-pipeline-test-default-pool-63948035-6541   Ready    <none>   33d    v1.24.8-gke.2000
```

### PipelineRun pods running in same node
Create the basic pipeline (and pipelineRun ) with four task:
```bash
$ kubectl apply -f examples/four-tasks-pipeline.yaml
pipeline.tekton.dev/demo-pipeline-four-tasks configured

$ kubectl create -f examples/four-tasks-pipelinerun.yaml 
pipelinerun.tekton.dev/demo-pipeline-four-taskspgkdf created
```

Check that all the pipeline task pods are scheduled to the same node:
```bash
$ kubectl get pods -o wide
NAME                                                READY   STATUS      RESTARTS   AGE   IP           NODE                                                  NOMINATED NODE   READINESS GATES
demo-pipeline-four-taskspgkdf-say-hello-again-pod   0/1     Completed   0          67s   10.60.5.17   gke-tekton-pipeline-test-default-pool-55a03fc3-t4jh   <none>           <none>
demo-pipeline-four-taskspgkdf-say-hello-pod         0/1     Completed   0          67s   10.60.5.16   gke-tekton-pipeline-test-default-pool-55a03fc3-t4jh   <none>           <none>
demo-pipeline-four-taskspgkdf-say-world-again-pod   0/1     Completed   0          67s   10.60.5.18   gke-tekton-pipeline-test-default-pool-55a03fc3-t4jh   <none>           <none>
demo-pipeline-four-taskspgkdf-say-world-pod         0/1     Completed   0          67s   10.60.5.15   gke-tekton-pipeline-test-default-pool-55a03fc3-t4jh   <none>           <none>
```

### PipelineRun repels with each other by Anti-Pod Affinity
This prototype is currently implemented with Anti-Pod Affinity configured for the placeholder pods.
In other words, different PipelineRuns are expected to be scheduled to different nodes.

Create multiple long running pipelines:
```bash
$ kubectl apply -f examples/long-running-pipeline.yaml 
pipeline.tekton.dev/long-running-demo-pipeline created

$ kubectl create -f examples/long-running-pipelinerun.yaml 
pipelinerun.tekton.dev/long-running-demo-pipelinerun-hd42r created
$ kubectl create -f examples/long-running-pipelinerun.yaml
pipelinerun.tekton.dev/long-running-demo-pipelinerun-dl9l8 created
$ kubectl create -f examples/long-running-pipelinerun.yaml
pipelinerun.tekton.dev/long-running-demo-pipelinerun-bp2ts created
$ kubectl create -f examples/long-running-pipelinerun.yaml
pipelinerun.tekton.dev/long-running-demo-pipelinerun-g8ndd created
```

Check that each PipelineRun is scheduled to separate nodes (removed placeholder pods from log for readability):
```bash
$ kubectl get pods -o wide
NAME                                            READY   STATUS    RESTARTS   AGE   IP            NODE                                                  NOMINATED NODE   READINESS GATES
long-running-demo-pipelinerun-bp2ts-sleep-pod   1/1     Running   0          43s   10.60.1.24    gke-tekton-pipeline-test-default-pool-55a03fc3-45dl   <none>           <none>
long-running-demo-pipelinerun-dl9l8-sleep-pod   1/1     Running   0          45s   10.60.0.210   gke-tekton-pipeline-test-default-pool-07b93c60-2xbi   <none>           <none>
long-running-demo-pipelinerun-g8ndd-sleep-pod   1/1     Running   0          42s   10.60.4.121   gke-tekton-pipeline-test-default-pool-63948035-6541   <none>           <none>
long-running-demo-pipelinerun-hd42r-sleep-pod   1/1     Running   0          46s   10.60.5.20    gke-tekton-pipeline-test-default-pool-55a03fc3-t4jh   <none>           <none>
```

When no nodes can met the scheduling predicate due to Anti-Pod Affinity, the cluster AutoScaler should be triggered
automatically (when configured) to create nodes up to the node pool size. To demonstrate, we create a N+1 PipelineRun (where N is your number of nodes) before the previous N PipelineRuns are completed

```bash
$ kubectl create -f examples/long-running-pipelinerun.yaml 
pipelinerun.tekton.dev/long-running-demo-pipelinerun-975bm created
```

The placeholder pod for this pipeline should trigger the cluster AutoScaler:

``` bash
$ kubectl describe pod custom-pod-affinity-031519cdb2-0
...
Events:
  Type     Reason            Age   From                Message
  ----     ------            ----  ----                -------
  Warning  FailedScheduling  2m5s  default-scheduler   0/4 nodes are available: 4 node(s) didn't match pod anti-affinity rules. preemption: 0/4 nodes are available: 4 No preemption victims found for incoming pod.
  Normal   TriggeredScaleUp  117s  cluster-autoscaler  pod triggered scale-up: [{https://www.googleapis.com/compute/v1/projects/zhangquan-test/zones/us-central1-c/instanceGroups/gke-tekton-pipeline-test-default-pool-63948035-grp 1->2 (max: 3)}]
  Normal   Scheduled         9s    default-scheduler   Successfully assigned default/custom-pod-affinity-031519cdb2-0 to gke-tekton-pipeline-test-default-pool-63948035-w4ng
  Normal   Pulling           7s    kubelet             Pulling image "gcr.io/zhangquan-test/nop-8eac7c133edad5df719dc37b36b62482@sha256:1e9f6b2919ec3efe251ab922820edfac97c736376d8e739b6108323e1097956d"
  Normal   Pulled            2s    kubelet             Successfully pulled image "gcr.io/zhangquan-test/nop-8eac7c133edad5df719dc37b36b62482@sha256:1e9f6b2919ec3efe251ab922820edfac97c736376d8e739b6108323e1097956d" in 5.444150619s
  Normal   Created           2s    kubelet             Created container affinity-assistant
  Normal   Started           1s    kubelet             Started container affinity-assistant
```

And you should see the N+1 node:

```
$ kubectl get nodes
NAME                                                  STATUS   ROLES    AGE     VERSION
gke-tekton-pipeline-test-default-pool-07b93c60-2xbi   Ready    <none>   33d     v1.24.8-gke.2000
gke-tekton-pipeline-test-default-pool-55a03fc3-45dl   Ready    <none>   179m    v1.24.8-gke.2000
gke-tekton-pipeline-test-default-pool-55a03fc3-t4jh   Ready    <none>   160m    v1.24.8-gke.2000
gke-tekton-pipeline-test-default-pool-63948035-6541   Ready    <none>   33d     v1.24.8-gke.2000
gke-tekton-pipeline-test-default-pool-63948035-w4ng   Ready    <none>   6m10s   v1.24.8-gke.2000
```
