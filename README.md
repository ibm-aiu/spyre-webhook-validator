# Webhook Validator for Spyre pods

This webhook validator will validate the following case(s):

- a pod requests or limits Spyre devices from more than one resource pool

This webhook is applied to CREATE event of `Pod`, `Job`, and `Deployment`.

## Steps

0. Prepare
    - Replace image in Makefile and in validator if needed.
    - Create `spyre-operator` namespace if needed

1. Build image

    ```bash
    make docker-build
    ```

2. Push image

    ```bash
    make docker-push
    ```

3. Deploy
    > Please create namespace `spyre-operator` and custom resource definition `spyreclusterpolicies.spyre.ibm.com` first if not exists

    ```bash
    make deploy
    ```

4. Wait until the pod is running. (All resources are deployed in spyre-operator namespace which is not a good practice. Please change as needed.)

    ```bash
    # kubectl get po

    NAME                                     READY   STATUS    RESTARTS   AGE
    spyre-webhook-validator-65d8bd9756-sm957   1/1     Running   0          18m
    ```

5. Try deploying resources in the testing folder.

    invalid resources:

    ```bash
    # oc create -f test/manifest/pod/invalid
    Error from server: error when creating "test/manifest/pod/invalid/deploy.yaml": admission webhook "spyre-deployment-validator.k8s.io" denied the request: A pod cannot request or limit Spyre from more than one resource pool.
    Error from server: error when creating "test/manifest/pod/invalid/job.yaml": admission webhook "spyre-job-validator.k8s.io" denied the request: A pod cannot request or limit Spyre from more than one resource pool.
    Error from server: error when creating "test/manifest/pod/invalid/pod.yaml": admission webhook "spyre-pod-validator.k8s.io" denied the request: A pod cannot request or limit Spyre from more than one resource pool

    # oc create -f test/manifest/clusterpolicy/invalid
    Error from server: error when creating "test/manifest/clusterpolicy/invalid/no-device-plugin.yaml": admission webhook "spyre-cluster-policy-validator.k8s.io" denied the request: failed to validate device plugin: failed to validate config: image not specified
    Error from server: error when creating "test/manifest/clusterpolicy/invalid/no-feature-discovery.yaml": admission webhook "spyre-cluster-policy-validator.k8s.io" denied the request: failed to validate feature discovery: failed to validate config: image not specified
    Error from server: error when creating "test/manifest/clusterpolicy/invalid/no-metrics-exporter.yaml": admission webhook "spyre-cluster-policy-validator.k8s.io" denied the request: failed to validate exporter: failed to validate config: image not specified
    Error from server: error when creating "test/manifest/clusterpolicy/invalid/no-pod-validator.yaml": admission webhook "spyre-cluster-policy-validator.k8s.io" denied the request: failed to validate validator: failed to validate config: image not specified
    Error from server: error when creating "test/manifest/clusterpolicy/invalid/no-scheduler.yaml": admission webhook "spyre-cluster-policy-validator.k8s.io" denied the request: failed to validate scheduler: failed to validate config: image not specified
    ```

    valid resources:

    ```bash
    # oc create -f test/manifest/pod/valid
    deployment.apps/valid-deployment created
    job.batch/valid-job created
    pod/valid-pod created

    # oc create -f test/manifest/clusterpolicy/valid
    spyreclusterpolicy.spyre.ibm.com/spyreclusterpolicy-minimum created
    ```

## Cleanup

```bash
make undeploy
# if valid resources were deployed, also run
kubectl delete -f test/manifest/pod/valid
kubectl delete -f test/manifest/clusterpolicy/valid
```

## Repository links

- [Contributing](CONTRIBUTING.md)
- [Maintainers](MAINTAINERS.md)
- [License](LICENSE)
