# PROMPTS.md — Challenge 5: Deploy to Kubernetes

This file documents the AI prompts used to help complete this challenge.

## Prompt 1 — Generate Jenkinsfile
**Prompt:**
> Create a Jenkins declarative pipeline with three stages: Lint, Build & Push, and Deploy. The Deploy stage should apply pod.yaml to a Kubernetes cluster using a token stored in a Jenkins credential called KUBECONFIG_TOKEN, then wait for the pod to be Ready.

**What it produced:**
A Jenkinsfile with `withCredentials` wrapping kubectl commands using `--token` and `--insecure-skip-tls-verify=true`.

---

## Prompt 2 — Generate pod.yaml with health probes
**Prompt:**
> Create a Kubernetes Pod manifest for an app named myapp using image ttl.sh/abhinavsiddharth007:2h, listening on port 4444. Include both a livenessProbe and a readinessProbe using HTTP GET on path / and port 4444.

**What it produced:**
A pod.yaml with both `livenessProbe` and `readinessProbe` configured with appropriate `initialDelaySeconds` and `periodSeconds`.

---

## Prompt 3 — Debug pod not reaching Ready
**Prompt:**
> My kubectl wait command is timing out and the pod shows ImagePullBackOff. The image is on ttl.sh. What could be wrong?

**What it helped with:**
Identified that the ttl.sh image TTL had expired. Solution: re-push the image with a fresh TTL before running the pipeline.

---

## Prompt 4 — Explain liveness vs readiness probes
**Prompt:**
> What is the difference between a livenessProbe and a readinessProbe in Kubernetes, and when should I use each?

**What it helped with:**
- **livenessProbe**: Restarts the container if it becomes unresponsive (deadlock, hung process).
- **readinessProbe**: Controls whether the pod receives traffic — keeps it out of service endpoints until it's ready to serve requests.
