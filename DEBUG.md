# DEBUG.md — Challenge 5: Deploy to AWS EC2

## Scenario
Pipeline shows SUCCESS. `curl localhost:4444` on the EC2 instance returns the expected JSON.
From your laptop, `curl http://<public-ip>:4444` **hangs forever** (not "connection refused" — it hangs until Ctrl-C).

The symptom shape is the whole game here: a **hang** means our SYN is being **silently dropped** somewhere and no response ever comes back. A **dropped** packet produces a hang; a packet that *reaches* a host with nothing listening produces a TCP **RST** → an immediate **"connection refused"**. So whatever the cause is, it has to be something that *drops* packets — that rules out a few tempting-but-wrong answers (see below).

---

## 1. Two Ranked Hypotheses

### Hypothesis 1 (Most Likely) — Security Group has no inbound rule for TCP/4444
The EC2's Security Group is missing an inbound rule for port 4444 (or the rule targets the wrong CIDR / protocol).
**Why a hang, not a refusal:** A Security Group is a stateful firewall that **drops** non-matching inbound packets at the AWS network boundary, before they ever reach the instance's OS. The SYN goes out, nothing comes back, and the client retransmits until it times out — that is the hang. Default SGs deny all inbound except what you explicitly allow, so a forgotten 4444 rule is the single most common cause.

### Hypothesis 2 (Less Likely) — A host-level firewall (or subnet NACL) is DROPping inbound 4444
Even with the SG open, a firewall *on the instance* — `ufw` / `iptables` / `nftables` / `firewalld` — or a restrictive subnet **Network ACL** can drop the packet.
**Why a hang, not a refusal:** A firewall rule with a **DROP** (not **REJECT**) target discards the packet without sending anything back — same indefinite hang as the SG case. (A **REJECT** target *would* send an ICMP/RST and give "connection refused", which we are not seeing — so if there's a host firewall here, it's configured to drop.) NACLs are also stateless and silently drop, though the default NACL allows all, which is why this ranks below the SG.

### Ruled out by the symptom — app bound to `127.0.0.1` instead of `0.0.0.0`
A common guess is that the app is listening only on the loopback interface. That is a real failure mode, but it does **not** match this symptom: when an external packet reaches the instance on a port with no matching listening socket, the kernel replies with a TCP **RST**, so the client sees an immediate **"connection refused"** — not a hang. Because we observe a hang, the packet is being dropped *before* it reaches a socket, which points at a firewall layer, not the bind address. (We still verify the bind address below, because it's cheap and it's the thing that bites *after* you open the SG.)

---

## 2. Verification Steps

### Verify Hypothesis 1 — Check the Security Group
Console: EC2 → Instances → select instance → **Security** tab → Inbound rules. Look for Type `Custom TCP`, Port `4444`, Source `0.0.0.0/0` (or your intended CIDR). Missing → that's it.

CLI:
```bash
aws ec2 describe-security-groups \
  --group-ids <sg-id> \
  --query 'SecurityGroups[*].IpPermissions' \
  --output table
```

### Verify Hypothesis 2 — Check the host firewall and the NACL
SSH into the instance and inspect every local firewall layer:
```bash
sudo ufw status                 # Ubuntu's frontend, if installed
sudo iptables -S                # raw rules
sudo nft list ruleset 2>/dev/null   # nftables, if used
```
Look for a DROP affecting tcp/4444. Then in the console check the subnet's **Network ACL** (VPC → Network ACLs → the one associated with the instance's subnet) for a deny/limited inbound or a missing ephemeral-port outbound rule.

### Cheap sanity check — confirm the bind address (rules in/out the "ruled out" cause)
```bash
sudo ss -tlnp | grep 4444
```
Good: `0.0.0.0:4444` or `*:4444` (reachable once the firewall is open).
If it shows `127.0.0.1:4444`, external clients would get **connection refused**, not a hang — a *different* bug to fix next, by changing the app's listen address.

---

## 3. The Fix

**If Hypothesis 1 (SG):** add an inbound rule — Type `Custom TCP`, Port `4444`, Source `0.0.0.0/0` (or narrower). No reboot needed; SG changes take effect immediately on new connections.

**If Hypothesis 2 (host firewall / NACL):** allow tcp/4444 at the offending layer, e.g. `sudo ufw allow 4444/tcp`, or add the matching `iptables -A INPUT -p tcp --dport 4444 -j ACCEPT` rule above the DROP, or fix the NACL inbound rule (and remember NACLs are stateless — the ephemeral-port range must be allowed *outbound* too).

Stay minimal — no Terraform rewrite, no new VPC. Adjusting one SG/firewall/NACL rule is the whole fix.

---

## 4. Underlying Lesson

**Dropped packet vs. packet at a closed port:**
- A **dropped** packet (Security Group / NACL / host-firewall DROP) gets *no response at all*. The client retransmits its SYN and eventually gives up — you experience this as a **hang/timeout**. The drop happens *before* the packet reaches a listening socket, so the firewall is invisible to the OS's port logic.
- A packet that **reaches the host** but hits a port with no listening socket gets an immediate TCP **RST** from the kernel — the client reports **"connection refused"** at once.

So the symptom is diagnostic: **hang ⇒ a filtering layer is silently dropping packets**; **refused ⇒ the port is reachable but nothing is listening** (app down, or bound to the wrong interface). They live at different layers, and the error string tells you which layer to go look at first.
