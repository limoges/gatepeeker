# Gatepeeker
>  Validate your resources against policies before they hit the cluster.

Gatepeeker is a tool designed to bridge the gap between [Open Policy Agent]'s ([OPA]) [Gatekeeper] validation running 
on the cluster, and configuration being developed in git repositories.

It is designed to help detect policy noncompliance early in the configuration development process.

**Features**
- Distribute Gatekeeper policies outside Kubernetes
- Check policy compliance anywhere
- Integrates well into CI/CD

**What differentiate Gatepeeker from...**

- [gator] focuses on policy creators by adding tests and testsuites to help with policy development.
    Gatepeeker takes these policies and allows validating compliance before resources reach the cluster.
- [gatekeeper-conftest] rewrites OPA inputs to admission requests, which is the schema [Gatekeeper], as an
    admission controller sees. It however doesn't support reading policies from [Gatekeeper] policies.

**Example**

Here's a quick example to validate your manifest against policies:
```bash
# Extract policies from rendered manifests. This could be `helm template .` or `argocd app manifests`...
cat k8s.yaml | gatepeeker build > policies.yaml

# Check deployment.yaml against the policies.
cat deployment.yaml | gatepeeker --policies policies.yaml
```

## Getting Started

## Installation

Gatepeeker is available for Linux, MacOS and Windows on the [release page].

Or, using go:
```bash
go install github.com/limoges/gatepeeker@latest
```
## Usage

The common workflow would be to package policies into a single artifact using `gatepeeker build`.
Then check for policy compliance with ` validate`.

### Example 1. Local file policies
```bash
# Scan k8s.yaml for Gatekeeper objects, like Constraints & ConstraintTemplates.
cat k8s.yaml | gatepeeker build > policies.yaml

# Check deployment.yaml against the policies defined in policies.yaml
gatepeeker --policies policies.yaml deployment.yaml
```
### Example 2. Validate local manifests against remote policies
```bash
cat deployment.yaml | gatepeeker validate \
    --policies https://raw.githubusercontent.com/open-policy-agent/-library/refs/heads/master/library/general/containerlimits/template.yaml
```
### Example 3. Pick policy changes from a GitOps source, like an ArgoCD App
```bash
$ argocd app login argocd.clusterx.example
$ argocd app platform-gatekeeper-policies | gatepeeker build > clusterx-policies.yaml   # Extract policies from an ArgoCD App
$ gatepeeker validate --policies clusterx-policies.yaml my-manifests.yaml               # Validate against the freshly extracted policies
```

# Thoughts

## Limitations
- There is currently only support for Validation using policies defined as custom resources, `ConstraintTemplates` and `Constraints`. No mutation support.

## Improvement Ideas
- Add support for loading templates + constraints from a "test..sh/v1alpha1/Suite".
- Explore alternative targets to admission.k8s..sh
- Show constraint + resource matches / no matches
- Add CTRF-formatted reporting to play nice in CICD
- Add remote reporting functionality so policy creators can get feedback on new policies impact in CICD.

[release page]:https://github.com/limoges/gatepeeker/releases
[gator]:https://open-policy-agent.github.io/gatekeeper/website/docs/gator/
[gatekeeper-conftest]:https://github.com/clover/gatekeeper-conftest
[Open Policy Agent]:https://www.openpolicyagent.org/
[OPA]:https://www.openpolicyagent.org/
[Gatekeeper]:https://github.com/open-policy-agent/gatekeeper
