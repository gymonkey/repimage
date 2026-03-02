# Namespace and Repository Filtering Design

## Overview

Add namespace and repository filtering capabilities to repimage, allowing users to control which Pods and images should be processed for mirror replacement.

## Requirements

1. Add `--namespaces` parameter to specify which namespaces to process
2. Add `--repositories` parameter to specify which image repositories to process
3. Both parameters must be set for any processing to occur
4. Use AND logic when both filters are active

## Behavior

### Core Rules

- **Both parameters must be set** for any image replacement to occur
- If either parameter is not set, no Pods are processed
- When both are set, Pod must satisfy BOTH conditions:
  - Pod's namespace is in the `--namespaces` list
  - Image repository is in the `--repositories` list

### Configuration Examples

**Enable filtering (both parameters required):**
```yaml
- /repimage
  - --namespaces=kube-system,default
  - --repositories=k8s.gcr.io,gcr.io
```
This will only process Pods in `kube-system` or `default` namespaces that use images from `k8s.gcr.io` or `gcr.io`.

**No processing (partial configuration):**
```yaml
- /repimage
  - --namespaces=kube-system
```
No processing - only `--namespaces` is set, `--repositories` is missing.

```yaml
- /repimage
  - --repositories=k8s.gcr.io
```
No processing - only `--repositories` is set, `--namespaces` is missing.

### Processing Flow

```
Admission Request Received
    ↓
Are --namespaces AND --repositories both set?
    ↓ No
    └─→ Return Allowed = true (no processing)
    ↓ Yes
Is Pod's namespace in --namespaces list?
    ↓ No
    └─→ Return Allowed = true (no processing)
    ↓ Yes
Is image repository in --repositories list?
    ↓ No
    └─→ Keep original image
    ↓ Yes
Apply --ignore-domains filtering
Apply skip annotation check
    ↓
Execute image replacement
```

## Implementation Changes

### main.go

Add new flag variables:
```go
namespaces = flag.String("namespaces", "", "Comma-separated list of namespaces to process")
repositories = flag.String("repositories", "", "Comma-separated list of repositories to process")
```

### pkg/utils/pods.go

Update `AdmitPods` function signature to accept namespace and repository lists, and add filtering logic.

### pkg/utils/parse.go

Add repository extraction and matching function.

## Compatibility

- Existing `--ignore-domains` parameter continues to work as before
- Existing `repimage.kubernetes.io/skip` annotation continues to work as before
- New filtering occurs before these existing checks
