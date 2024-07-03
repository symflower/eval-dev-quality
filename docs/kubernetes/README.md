# Running "eval-dev-quality" on Kubernetes


### Prerequisite Checklist

- `kubectl` installed and configured with authentication to the cluster.
- Dedicated Namespace to run the jobs.
- RWX volume to store the evaluation results (check `volume.yml` for inspiration).

### Running a evaluation without the eval-dev-quality kubernetes runtime

- Change the `command` in `job.yml` to the desired command.
- Run it with `kubectl --namespace $NAMESPACE apply -f job.yml`.
- Check the job with `kubectl get pods --namespace $NAMESPACE` until status shows `completed`.
- Remove the job with `kubectl --namespace $NAMESPACE delete --force -f job.yml`.
- [Getting the evaluation data](#getting-the-evaluation-data)

### Running multiple evaluations with the eval-dev-quality kubernetes runtime

- Define all the models with `--model` which should be run inside the containerized workload.
- Define the parameter `--runtime kubernetes` to indicate that jobs should run inside a kubernetes cluster.
- Define the parameter `--parallel 20` to indicate how many jobs should run simultaneously.
- [Getting the evaluation data](#getting-the-evaluation-data)

Example:
```bash
eval-dev-quality evaluate --runtime kubernetes --runs 5 --model symflower/symbolic-execution --model symflower/symbolic-execution --model symflower/symbolic-execution --repository golang/plain --parallel 2
```
This commands run 3x the `symflower/symbolic-execution` model with 5 runs of each model inside a containerized workload on the kubernetes cluster, it will limit the parallel execution to 2 containers at the same time.

### Getting the evaluation data

- Run `kubectl --namespace $NAMESPACE apply -f connect.yml` to start a busybox container with the PVC mount.
- Use `kubectl cp eval-dev-quality/evaluation-storage-access-XXXXXXX:/var/evaluations ./evaluations` to copy all evaluations to a local folder.
- Delete the pod with `kubectl --namespace $NAMESPACE delete --force -f connect.yml` when not needed anymore.
