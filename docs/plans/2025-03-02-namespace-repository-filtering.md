# Namespace and Repository Filtering Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add namespace and repository filtering to repimage, allowing users to control which Pods and images are processed for mirror replacement.

**Architecture:** Add two new command-line flags (`--namespaces` and `--repositories`) to main.go, pass them through to the AdmitPods function, and add filtering logic before image replacement. Both parameters must be set for any processing to occur, using AND logic.

**Tech Stack:** Go 1.x, Kubernetes admission webhook, flag package, strings package

---

### Task 1: Add new flag variables to main.go

**Files:**
- Modify: `main.go:16-21`

**Step 1: Add the new flag declarations**

Add after line 20:

```go
namespaces   = flag.String("namespaces", "", "Comma-separated list of namespaces to process (required)")
repositories = flag.String("repositories", "", "Comma-separated list of repositories to process (required)")
```

**Step 2: Verify code compiles**

Run: `go build -o /tmp/repimage`
Expected: SUCCESS (no errors)

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add namespaces and repositories flag declarations

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 2: Add repository extraction function to pkg/utils/parse.go

**Files:**
- Modify: `pkg/utils/parse.go`

**Step 1: Write the failing test**

Add to `pkg/utils/parse_test.go`:

```go
func TestExtractRepository(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{
			name:     "simple image without domain",
			image:    "nginx",
			expected: "docker.io",
		},
		{
			name:     "image with user/repo",
			image:    "library/nginx",
			expected: "docker.io",
		},
		{
			name:     "image with domain",
			image:    "k8s.gcr.io/coredns/coredns",
			expected: "k8s.gcr.io",
		},
		{
			name:     "image with three parts",
			image:    "gcr.io/my-project/my-image",
			expected: "gcr.io",
		},
		{
			name:     "legacy docker.io domain",
			image:    "index.docker.io/library/nginx",
			expected: "docker.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRepository(tt.image)
			if result != tt.expected {
				t.Errorf("ExtractRepository(%q) = %q, want %q", tt.image, result, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/utils -v -run TestExtractRepository`
Expected: FAIL with "undefined: ExtractRepository"

**Step 3: Write minimal implementation**

Add to `pkg/utils/parse.go`:

```go
// ExtractRepository extracts the repository/domain from an image reference
func ExtractRepository(image string) string {
	parts := strings.SplitN(image, "/", 3)
	switch len(parts) {
	case 1:
		// nginx -> docker.io
		return defaultDomain
	case 2:
		// user/repo -> docker.io (if not a domain)
		// domain/repo -> domain
		if !isDomain(parts[0]) {
			return defaultDomain
		}
		if isLegacyDefaultDomain(parts[0]) {
			return defaultDomain
		}
		return parts[0]
	case 3:
		// domain/user/repo -> domain
		// user/repo/tag -> docker.io (if not a domain)
		if !isDomain(parts[0]) {
			return defaultDomain
		}
		if isLegacyDefaultDomain(parts[0]) {
			return defaultDomain
		}
		return parts[0]
	}
	return defaultDomain
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/utils -v -run TestExtractRepository`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/utils/parse.go pkg/utils/parse_test.go
git commit -m "feat: add ExtractRepository function

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 3: Update AdmitPods function signature in pkg/utils/pods.go

**Files:**
- Modify: `pkg/utils/pods.go:24-25`

**Step 1: Update function signature**

Change:
```go
func AdmitPods(prefix string, ignoreDomains []string, ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
```

To:
```go
func AdmitPods(prefix string, ignoreDomains []string, namespaces []string, repositories []string, ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
```

**Step 2: Verify code compiles**

Run: `go build -o /tmp/repimage`
Expected: FAIL with "not enough arguments" in main.go

**Step 3: Commit**

```bash
git add pkg/utils/pods.go
git commit -m "feat: update AdmitPods signature with namespace and repository filters

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 4: Add namespace filtering logic to AdmitPods

**Files:**
- Modify: `pkg/utils/pods.go`

**Step 1: Add namespace filtering after the annotation check**

Add after line 54 (after the annotation check block):

```go
	// Check if both namespaces and repositories are configured
	if len(namespaces) == 0 || len(repositories) == 0 {
		klog.Info("namespaces or repositories not configured, skipping image rewriting")
		reviewResponse := admissionv1.AdmissionResponse{}
		reviewResponse.Allowed = true
		return &reviewResponse
	}

	// Check if the pod's namespace is in the allowed list
	namespaceAllowed := false
	for _, ns := range namespaces {
		if pod.Namespace == ns {
			namespaceAllowed = true
			break
		}
	}
	if !namespaceAllowed {
		klog.Infof("pod namespace %q not in allowed list, skipping image rewriting", pod.Namespace)
		reviewResponse := admissionv1.AdmissionResponse{}
		reviewResponse.Allowed = true
		return &reviewResponse
	}
```

**Step 2: Verify code compiles**

Run: `go build -o /tmp/repimage`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add pkg/utils/pods.go
git commit -m "feat: add namespace filtering logic

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 5: Add repository filtering logic in container loop

**Files:**
- Modify: `pkg/utils/pods.go:59-66`

**Step 1: Modify the container loop to check repository**

Replace the existing container loop (lines 59-66):

```go
	reviewResponse := admissionv1.AdmissionResponse{}
	containers := pod.Spec.Containers

	var updated bool
	for i, container := range containers {
		repo := ExtractRepository(container.Image)
		repoAllowed := false
		for _, allowedRepo := range repositories {
			if repo == allowedRepo {
				repoAllowed = true
				break
			}
		}

		if !repoAllowed {
			klog.Infof("image repository %q not in allowed list, skipping image %q", repo, container.Image)
			continue
		}

		newImage := ReplaceImageName(prefix, ignoreDomains, container.Image)
		if newImage != container.Image {
			containers[i].Image = newImage
			updated = true
		}
	}
```

**Step 2: Verify code compiles**

Run: `go build -o /tmp/repimage`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add pkg/utils/pods.go
git commit -m "feat: add repository filtering in container loop

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 6: Update servePods function in main.go

**Files:**
- Modify: `main.go:64-74`

**Step 1: Update servePods to parse and pass new parameters**

Replace the entire `servePods` function:

```go
func servePods(w http.ResponseWriter, r *http.Request) {
	var domains []string
	if *ignoreDomains != "" {
		domains = strings.Split(*ignoreDomains, ",")
		// Trim whitespace from each domain
		for i := range domains {
			domains[i] = strings.TrimSpace(domains[i])
		}
	}

	var nsList []string
	if *namespaces != "" {
		nsList = strings.Split(*namespaces, ",")
		// Trim whitespace from each namespace
		for i := range nsList {
			nsList[i] = strings.TrimSpace(nsList[i])
		}
	}

	var repoList []string
	if *repositories != "" {
		repoList = strings.Split(*repositories, ",")
		// Trim whitespace from each repository
		for i := range repoList {
			repoList[i] = strings.TrimSpace(repoList[i])
		}
	}

	serve(w, r, *prefix, domains, nsList, repoList)
}
```

**Step 2: Update serve function signature**

Change the `serve` function signature (line 23):

```go
func serve(w http.ResponseWriter, r *http.Request, prefix string, ignoreDomains []string, namespaces []string, repositories []string) {
```

And update the call to `AdmitPods` (line 47):

```go
		resAdmissionReview.Response = utils.AdmitPods(prefix, ignoreDomains, namespaces, repositories, reqAdmissionReview)
```

**Step 3: Verify code compiles**

Run: `go build -o /tmp/repimage`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: wire up namespace and repository parameters

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 7: Update README documentation

**Files:**
- Modify: `README.md`

**Step 1: Add new configuration section**

Add after line 35 (after "忽略指定域名" section):

```markdown
## 指定 Namespace 和仓库

默认情况下，不设置参数时不处理任何 Pod。必须同时设置 `--namespaces` 和 `--repositories` 参数才会启用镜像替换功能。

- `--namespaces`: 指定需要处理的 namespace 列表（逗号分隔）
- `--repositories`: 指定需要处理的镜像仓库列表（逗号分隔）

例如：
\`\`\`yaml
containers:
  - command:
      - /repimage
      - --namespaces=kube-system,default
      - --repositories=k8s.gcr.io,gcr.io,quay.io
\`\`\`

这样配置后，只有在 `kube-system` 或 `default` namespace 中使用 `k8s.gcr.io`、`gcr.io` 或 `quay.io` 仓库镜像的容器才会被替换。

**注意：**
- 两个参数必须同时设置才会生效
- 任意一个参数未设置，则不处理任何 Pod
- 使用 AND 逻辑：Pod 必须同时满足 namespace 和仓库条件
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add namespace and repository filtering documentation

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 8: Run all tests

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 2: Build final binary**

Run: `go build -o repimage`
Expected: SUCCESS

**Step 3: Verify help output**

Run: `./repimage --help`
Expected: See `--namespaces` and `--repositories` in flag list

**Step 4: Final commit if needed**

If any adjustments were needed:

```bash
git add .
git commit -m "test: ensure all tests pass and build succeeds

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```
