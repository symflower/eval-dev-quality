# Running "eval-dev-quality" on Kubernetes

### Prerequisite Checklist

- `kubectl` installed and configured with authentication to the cluster.
- Dedicated Namespace to run the jobs.
- RWX volume to store the evaluation results (check `volume.yml` for inspiration).

### Running multiple evaluations with the eval-dev-quality kubernetes runtime

- Define all the models with `--model` which should be run inside the containerized workload.
- Define the parameter `--runtime kubernetes` to indicate that jobs should run inside a kubernetes cluster.
- Define the parameter `--parallel 20` to indicate how many jobs should run simultaneously.

Example:
```bash
eval-dev-quality evaluate --runtime kubernetes --runs 5 --model symflower/symbolic-execution --model symflower/symbolic-execution --model symflower/symbolic-execution --repository golang/plain --parallel 2
```
This commands run 3x the `symflower/symbolic-execution` model with 5 runs of each model inside a containerized workload on the kubernetes cluster, it will limit the parallel execution to 2 containers at the same time.
