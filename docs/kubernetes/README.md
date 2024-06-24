# Running "eval-dev-quality" on Kubernetes


### Prerequisite Checklist

- Dedicated Namespace to run the jobs.
- RWX volume to store the evaluation results (check `volume.yml` for inspiration).

### Running a job

- Change the `command` in `job.yml` to the desired command.
- Run it with `kubectl --namespace $NAMESPACE apply -f job.yml`.
- Check the job with `kubectl get pods --namespace $NAMESPACE` until status shows `completed`.
- Remove the job with `kubectl --namespace $NAMESPACE delete --force -f job.yml`.

### Getting the evaluation data

- Run `kubectl --namespace $NAMESPACE apply -f connect.yml` to start a busybox container with the PVC mount.
- Use `kubectl cp eval-dev-quality/evaluation-storage-access-XXXXXXX:/var/evaluations ./evaluations` to copy all evaluations to a local folder.
- Delete the pod with `kubectl --namespace $NAMESPACE delete --force -f connect.yml` when not needed anymore.
