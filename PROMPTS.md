# PROMPTS.md — Challenge 5: Deploy to AWS EC2

This file documents the AI prompts used and the reasoning behind key decisions in this challenge.

---

## Prompt 1 — Generate the EC2 deploy Jenkinsfile
**Prompt:**
> Write a Jenkins declarative pipeline that builds a Go binary, copies it to an AWS EC2 instance via scp, SSHs in, installs it under systemd, and waits for it to respond on port 4444. The SSH private key should come from a Jenkins credential with ID EC2_SSH_KEY.

**What it produced:**
A Jenkinsfile with stages — Validate, Build, Deploy to EC2, and Verify. The Deploy stage uses `withCredentials([sshUserPrivateKey(...)])` to inject the `.pem` without writing it to the repo, then runs `scp` followed by a heredoc `ssh` session that writes the systemd unit file, reloads the daemon, and polls `curl localhost:4444` until the app is up.

**Key decisions surfaced by the prompt:**
- **Static cross-compile.** The Build stage pins `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`. t2.micro is amd64, and a static binary sidesteps both the "exec format error" arch trap and glibc-version mismatches when a bare binary is copied to a host that didn't build it.
- **Health-gate the deploy.** The local `curl` poll must actually fail the build (`exit 1`) if the app never answers; otherwise the pipeline reports SUCCESS while the app is down — the exact false-green that the debug scenario is built around.
- **Timeouts on the external curl.** A Security Group DROP makes an un-timed `curl` hang forever and stall the Jenkins executor, so the Verify stage uses `--connect-timeout`/`--max-time` and fails fast with a diagnostic message.
- **`StrictHostKeyChecking=no`** is a pragmatic playground choice; in production you'd pre-populate `known_hosts` or use AWS Instance Connect / SSM instead of open SSH.
- **IP as a build parameter**, not a committed constant — the public IP changes on stop/start unless an Elastic IP is attached.

---

## Prompt 2 — systemd unit file shape
**Prompt:**
> What should a minimal systemd unit file look like for a Go binary that listens on a port, with automatic restart on failure?

**What it produced:**
A unit with `Restart=always`, `RestartSec=5`, and output directed to `journal`. Using `After=network.target` ensures the binary doesn't start before the network stack is up — important for a service that immediately tries to bind a port.

---

## Prompt 3 — Debug: curl hangs instead of "connection refused"
**Prompt:**
> My Jenkins pipeline shows SUCCESS. ssh + curl localhost:4444 on the EC2 returns JSON. curl http://<public-ip>:4444 from my laptop hangs forever. It doesn't say "connection refused" — it just hangs. What are the most likely causes ranked by probability?

**What it helped with:**
The hang-vs-"connection refused" distinction is the key diagnostic, and it actively *rules things in and out*:
- **#1 hypothesis — Security Group DROP.** A dropped packet never gets a TCP RST back, so the client retransmits and times out — a hang. The default SG denies inbound, so a forgotten tcp/4444 rule is the usual culprit.
- **#2 hypothesis — a host-level firewall (ufw/iptables/nftables) or subnet NACL DROPping 4444.** A DROP target (as opposed to REJECT) also discards silently → the same hang.
- **Ruled out by the symptom — app bound to `127.0.0.1`.** A correction to my first instinct here: a loopback-only bind does *not* cause a hang. When an external packet reaches the instance on a port with no matching socket, the kernel sends a TCP **RST**, so the client gets an immediate **"connection refused"** — not a hang. (The "kernel silently drops without ICMP" reasoning is a UDP behavior, not TCP.) Because we observe a hang, the packet is being dropped *before* it reaches a socket — pointing at a firewall layer, not the bind address. The bind address is still worth a cheap `ss -tlnp` check, since it's the thing that bites right after you open the SG.

---

## Stretch Task — Tag the EC2 instance (Cohort + Owner tags)

### What I did
After launching the EC2 instance, I added two tags in the AWS console:
- `Cohort` = `CS411-2026`
- `Owner` = `AbhinavSiddharth007`

Tags are added via EC2 → Instances → select instance → Tags tab → Manage tags.

### Why real cloud teams tag every resource at creation time
**Cost attribution.** Without tags, an AWS bill for a team of 20 is a single number with no breakdown — you can't tell which project, environment, or owner is responsible for a spike. With `Owner` and `Cohort` tags, you can filter Cost Explorer by tag and immediately see who is running what. This matters most on free-tier accounts, where an untagged forgotten instance isn't caught until the bill arrives. Retrofitting tags always leaves gaps: instances launched before the policy, forgotten scratch environments, and auto-scaling children spawned untagged. The only reliable window to tag is at creation — which is why AWS Organizations Service Control Policies can deny `RunInstances` unless required tags are present. Missing tags at creation = permanently unknown cost and ownership.

### Prompt used for the tagging rationale
**Prompt:**
> What is one concrete reason a real cloud engineering team enforces resource tagging at instance creation time rather than retroactively?

**Reasoning surfaced:**
Retroactive tagging always has gaps, and you cannot reliably backfill what you don't remember creating. Tagging at creation (enforced via SCPs that block untagged `RunInstances`) is the only point where coverage is guaranteed.
