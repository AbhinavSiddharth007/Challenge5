# DEBUG.md — Challenge 5: Deploy to Kubernetes

## Issues Encountered & How They Were Resolved

### 1. Pod not reaching Ready state
**Symptom:** `kubectl wait pod/myapp --for=condition=Ready` times out.  
**Cause:** Image TTL expired on ttl.sh (images expire after the specified duration, e.g., 2h).  
**Fix:** Re-push the image to ttl.sh with a fresh TTL, then re-run the pipeline.

```bash
docker tag myapp ttl.sh/abhinavsiddharth007:2h
docker push ttl.sh/abhinavsiddharth007:2h
```

### 2. Jenkins credential not found
**Symptom:** Pipeline fails with `CredentialNotFoundException: KUBECONFIG_TOKEN`.  
**Cause:** Credential ID was entered incorrectly (e.g., extra space, wrong casing).  
**Fix:** Go to Jenkins → Credentials, delete and recreate with ID exactly `KUBECONFIG_TOKEN`.

### 3. kubectl: command not found in Jenkins
**Symptom:** `sh: kubectl: not found` in the Deploy stage.  
**Cause:** `kubectl` is not installed on the Jenkins agent.  
**Fix:** Install kubectl on the Jenkins agent or use the playground's built-in agent that has it pre-installed.

### 4. TLS certificate verification failure
**Symptom:** `Unable to connect to the server: x509: certificate signed by unknown authority`.  
**Cause:** Self-signed cert in the playground cluster.  
**Fix:** Use `--insecure-skip-tls-verify=true` flag in kubectl commands (already included in Jenkinsfile).

### 5. Liveness/Readiness probe causing CrashLoopBackOff
**Symptom:** Pod keeps restarting.  
**Cause:** Probe path `/` on port `4444` returns non-2xx or the app takes too long to start.  
**Fix:** Increase `initialDelaySeconds` or verify the app actually listens on port 4444.
