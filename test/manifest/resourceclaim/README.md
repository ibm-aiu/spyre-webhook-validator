# ResourceClaim webhook test (CRC)

Verifies the `spyre-resourceclaim-validator.k8s.io` webhook decisions. It only
checks whether a `ResourceClaim` is **allowed or denied at admission time** —
not whether a Pod actually gets a device.

All tests run in the **`spyre-apps`** namespace. `spyre-operator` (and the
platform namespaces) are excluded by the webhook's `namespaceSelector`, so the
webhook does not fire there and any DeviceClass is implicitly allowed — that
already satisfies "spyre-operator may request spyre-pf / spyre-privileged-vf".

## How the validator decides

The webhook reads three values that are cached in memory whenever a
`SpyreClusterPolicy` passes `/validate-clusterpolicy`:

- `draDriver` = `SpyreClusterPolicy.spec.devicePlugin.draDriver`
- `vfMode` = `SpyreClusterPolicy.spec.cardManagement.enabled`
- `operatorNamespace` = `SpyreClusterPolicy.status.namespace` (irrelevant here,
  because spyre-operator is excluded and spyre-apps never matches it)

**`draDriver` is the master switch.** When it is `false`, the classic device
plugin allocates Spyre and the webhook imposes **no** restriction — every claim
is allowed. When it is `true`, the DeviceClass rules below apply:

| DeviceClass | VF mode ON | VF mode OFF |
| --- | --- | --- |
| `spyre-pf` | denied in spyre-apps | allowed |
| `spyre-privileged-vf` | denied | denied |
| `spyre-standard-vf` | allowed | denied |

## Prerequisites

1. **DRA enabled** — `oc api-resources | grep resourceclaims` lists
   `resource.k8s.io`. On OpenShift this needs the TechPreviewNoUpgrade feature
   set. Set `apiVersion` in the claim manifests and the webhook rule to the
   version shown.
2. **CRD** `spyreclusterpolicies.spyre.ibm.com` installed. Generate it from the
   spyre-operator module already in go.mod and apply it:

   ```bash
   make deploy-crd     # generates into test/manifest/crd/ then oc apply
   # or generate only, without touching the cluster:
   make install-crd
   ```

3. **Webhook deployed** with the resourceclaim entry registered:
   `make deploy` (the entry was added to
   `test/manifest/webhook-config/validatingwebhookconfig.yaml`).

## Run

The six cases are driven by the Ginkgo suite in [`test/e2e`](../../e2e), aligned
with the unit tests. It runs against the current kubeconfig and is guarded by
the `e2e` build tag so `make test` never picks it up:

```bash
make deploy        # webhook must be running first
make e2e-test      # go test -tags e2e ./test/e2e/...
```

The suite creates the namespace, toggles VF mode by building the
SpyreClusterPolicy in Go, attempts each claim, and asserts allow/deny.

## Manual run

The manifests in this directory let you reproduce the same cases with `oc`
directly (useful for ad-hoc debugging):

```bash
oc apply -f spyre-apps-namespace.yaml

# DRA driver ON, VF mode ON
oc apply -f clusterpolicy/vf-on.yaml
oc create -f claim/standard-vf.yaml     # allowed
oc create -f claim/pf.yaml              # denied
oc create -f claim/privileged-vf.yaml   # denied

# DRA driver ON, VF mode OFF
oc apply -f clusterpolicy/vf-off.yaml
oc create -f claim/pf.yaml              # allowed
oc create -f claim/privileged-vf.yaml   # denied
oc create -f claim/standard-vf.yaml     # denied

# DRA driver OFF (classic device plugin): no restriction, all allowed
oc apply -f clusterpolicy/dra-off.yaml
oc create -f claim/pf.yaml              # allowed
oc create -f claim/privileged-vf.yaml   # allowed
oc create -f claim/standard-vf.yaml     # allowed
```

Denied requests return:
`admission webhook "spyre-resourceclaim-validator.k8s.io" denied the request: <reason>`

## Notes

- The webhook caches `draDriver` and `vfMode` in memory. If the validator Pod
  restarts, re-apply the `clusterpolicy/*.yaml` before testing so the cache is
  repopulated.
- The `SpyreClusterPolicy` must be named `spyreclusterpolicy` and otherwise
  valid, or `/validate-clusterpolicy` rejects it and the cache is not updated.
- DeviceClass objects need not exist; the webhook only inspects the requested
  DeviceClass name.

## Cleanup

```bash
oc delete -f claim/ --ignore-not-found
oc delete -f clusterpolicy/vf-off.yaml --ignore-not-found
oc delete -f spyre-apps-namespace.yaml --ignore-not-found
```
