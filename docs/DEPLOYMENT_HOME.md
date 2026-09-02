# Home-Server Deployment Runbook (laptop + Cloudflare Tunnel)

From a spare laptop to a live, HTTPS-served, auto-deploying shop for the
price of a domain (~$10/year). This is the deployment actually in use
(decision #100, reversing #12); [DEPLOYMENT.md](DEPLOYMENT.md) remains the
runbook for a future move to a VPS, and steps that are identical are
referenced there instead of repeated.

Commands marked 💻 run on your Windows machine, 🖥️ on the laptop.

## Why a tunnel, and what it changes — the concepts

**The constraint.** The ISP (LIKENET) parks the router inside their own
private network: its WAN interface holds `192.168.197.84`, not a public
address (checked 2026-09-02 — `Status → Network Interface → WAN` on the
ZTE F673AV9; the diagnosis chain is in the Learning Log). That is CGNAT:
inbound traffic dies at the ISP's NAT, so DNS + port forwarding — the
classic recipe, and decision #12's plan — cannot work at any price except
paying the ISP for a public IP. No IPv6 either.

**The inversion.** A Cloudflare *named tunnel* makes inbound reachability
unnecessary: the `cloudflared` container dials **out** to Cloudflare's
network and keeps those connections alive; when a visitor opens the site,
Cloudflare relays the request back down the already-open connection.
Nothing ever connects *to* the house — the ISP sees ordinary outgoing
traffic. (Same inversion Tailscale uses for our management plane; this is
the public-web version of it.)

**Where TLS terminates.** The visitor's browser does its handshake with
Cloudflare's edge, which presents a real, automatically-renewed
certificate for the domain — genuine padlock, no warnings. The second
leg, edge → laptop, rides inside the tunnel's own encrypted channel. So
Caddy and its Let's Encrypt duties are gone from the prod stack
(couldn't work anyway: the ACME challenge is an inbound connection);
`cloudflared` hands requests straight to nginx (`web:80`). The honest
trade, accepted in #100: Cloudflare decrypts and re-encrypts in the
middle — the "no third party in the traffic path" ideal of #12 is
exactly what CGNAT made impossible.

**The perimeter.** With no inbound ports, the firewall story collapses to
"SSH from LAN/tailnet only". There is no port forwarding section in this
runbook at all — the router stays untouched, and the home IP is never
published anywhere (a DNS lookup of the domain returns Cloudflare's
addresses, not yours).

## 0. What you need first

- The spare laptop (any 64-bit machine with 4GB+ RAM is plenty)
- A domain you own (~$10/year — Cloudflare Registrar sells at cost;
  Porkbun/Namecheap are fine too). DuckDNS-style free names can't be
  used: Cloudflare must host the domain's DNS.
- A free Cloudflare account and a free Tailscale account
- ~2 hours

## 1. Install Ubuntu Server 24.04 on the laptop

💻 Download the Ubuntu Server 24.04 LTS ISO, write it to a USB stick with
Rufus, boot the laptop from it. Through the installer:

- **Wired ethernet if at all possible** — Wi-Fi works, but it's one more
  thing to debug on a headless box.
- User: `deploy` (matches the CI secrets).
- ☑ **Install OpenSSH server** when offered.

After first boot, note the LAN address: 🖥️ `ip -4 addr` (something like
`192.168.1.60`). Verify 💻 `ssh deploy@192.168.1.60` works — from here on
the laptop can live closed in a corner.

## 2. Make a laptop behave like a server

A laptop is a server with a built-in UPS (the battery) — but its defaults
assume a human: close the lid and it sleeps, power returns after an
outage and it stays off. Undo both. 🖥️:

```bash
# Closing the lid must do nothing (logind is systemd's seat/session manager)
sudo sed -i 's/^#\?HandleLidSwitch=.*/HandleLidSwitch=ignore/' /etc/systemd/logind.conf
sudo sed -i 's/^#\?HandleLidSwitchExternalPower=.*/HandleLidSwitchExternalPower=ignore/' /etc/systemd/logind.conf
sudo systemctl restart systemd-logind

# The machine must never suspend/hibernate on its own; mask makes the
# units un-startable rather than merely disabled
sudo systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target
```

In the BIOS/UEFI (one-time, lid open): find **"Restore on AC power
loss" → Power On** (names vary: "AC Back", "Wake on AC") so the shop
comes back by itself after an outage outlasts the battery.

## 3. Hardening

🖥️:

```bash
sudo apt update && sudo apt install -y ufw fail2ban unattended-upgrades

# The tunnel is outbound-only, so the ONLY inbound service is SSH —
# reachable from the LAN and the tailnet, never from the internet.
sudo ufw allow OpenSSH
sudo ufw --force enable
```

Then key-only SSH, as in DEPLOYMENT.md step 2: put your 💻 public key in
`~/.ssh/authorized_keys`, set `PermitRootLogin no` and
`PasswordAuthentication no` in `/etc/ssh/sshd_config`, restart ssh — and
verify key login from a second terminal before closing the first.

## 4. Install Docker

Identical to DEPLOYMENT.md step 3. 🖥️:

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker deploy
# log out and back in, then verify:
docker ps
```

## 5. Tailscale — the management plane

The public web reaches the laptop through Cloudflare; *you and CI* reach
it through Tailscale. 💻 Create a tailnet at tailscale.com (free plan,
sign in with GitHub), then:

```bash
# 🖥️ laptop
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up          # prints a login URL — open it on 💻 and approve
tailscale status           # note the machine name, e.g. mb-server
```

💻 Install the Tailscale app on Windows too, sign into the same tailnet,
and verify `ssh deploy@mb-server` works — that address works from
anywhere, home or not, CGNAT notwithstanding.

CI pieces, in the Tailscale **admin console**:

1. **Access controls**: declare a tag for CI runners and let it reach
   SSH. Merge into the policy:

   ```jsonc
   "tagOwners": { "tag:ci": ["autogroup:admin"] },
   "acls": [
     // default policy allows all; once you tighten it, keep at least:
     { "action": "accept", "src": ["tag:ci"], "dst": ["*:22"] }
   ]
   ```

2. **Settings → OAuth clients → Generate**: scope `auth_keys`, tag
   `tag:ci`. Copy ID and secret — they become `TS_OAUTH_CLIENT_ID` /
   `TS_OAUTH_SECRET` in step 8. The deploy job uses them to join as an
   *ephemeral* node that evaporates when the job ends.

3. **Machines → the laptop → ⋯ → Disable key expiry** — otherwise the
   laptop silently drops off the tailnet in ~180 days and deploys start
   failing mysteriously.

## 6. Domain + Cloudflare

💻 All in the browser:

1. Buy the domain (skip if you already own one).
2. Cloudflare dashboard → **Add a domain** → Free plan. Cloudflare shows
   two nameservers.
3. At the registrar, replace the domain's nameservers with those two.
   (Bought via Cloudflare Registrar? Nothing to do.) Wait for the zone
   to show **Active** — minutes to hours.

From this point Cloudflare *is* the domain's DNS, which is what lets the
tunnel bind hostnames to it.

## 7. Create the named tunnel

💻 Cloudflare dashboard → **Zero Trust** → Networks → **Tunnels** →
Create a tunnel → type **Cloudflared** → name it `mountain-breath`.

1. On the connector page, pick any OS — you only need the **token**: the
   long string after `--token` in the shown command. Don't run their
   installer; our compose stack runs the connector. The token goes into
   `deploy/.env` in step 8.
2. **Public hostname** tab → Add:
   - Hostname: your domain, e.g. `mountainbreath.com` (subdomain empty
     for the apex; add a second entry for `www` later if wanted)
   - Service: **HTTP** → `web:80`

   `web:80` works because cloudflared runs *inside the compose network*,
   where `web` is nginx's DNS name — the same nginx that already routes
   `/api/*` to the Go API and everything else to the React build. The
   tunnel needs no knowledge of our routing; it delivers every request
   to nginx and nginx does what it always did.

Note where the routing lives: hostname→service mapping is **dashboard
config**, not a file in the repo. The compose service only carries the
token that says which tunnel it is.

## 8. Clone, configure, first boot

Same as DEPLOYMENT.md steps 4–6 with a tunnel-flavored `.env`. 🖥️:

```bash
# GHCR pull token (read:packages only) — DEPLOYMENT.md step 4b
docker login ghcr.io -u Nerses01

sudo mkdir -p /opt/mountain-breath && sudo chown deploy:deploy /opt/mountain-breath
git clone https://github.com/Nerses01/Mountain_Breath.git /opt/mountain-breath
cd /opt/mountain-breath/deploy

# hex, not base64: the password rides inside a URL
# (postgres://mb:PASSWORD@postgres/…), and base64's / + = characters
# break URL parsing. Hex is URL-safe by construction.
openssl rand -hex 24

cat > .env <<'EOF'
POSTGRES_USER=mb
POSTGRES_PASSWORD=<the generated hex string>
POSTGRES_DB=mountain_breath
DOMAIN=mountainbreath.com
CLOUDFLARE_TUNNEL_TOKEN=<token from step 7>
EOF
chmod 600 .env
# If you ever change POSTGRES_PASSWORD after the first boot: the postgres
# image only APPLIES the password when initializing an empty volume, so an
# edit here silently diverges from what the database expects. Before any
# data exists, `down -v` + up is the fix; after, change it inside Postgres.

cd /opt/mountain-breath
docker compose -f deploy/docker-compose.prod.yml pull
docker compose -f deploy/docker-compose.prod.yml up -d
docker compose -f deploy/docker-compose.prod.yml ps                 # wait: everything healthy
docker compose -f deploy/docker-compose.prod.yml logs cloudflared   # expect: "Registered tunnel connection" ×4
```

Checks, in order: 🖥️ `curl -s http://localhost:8081/health` proves the
stack itself (loopback smoke port, no internet involved); then 💻 open
`https://yourdomain.com` — padlock, no warnings, from anywhere. In the
Zero Trust tunnels list the tunnel shows **HEALTHY**.

Seed + admin promotion: exactly DEPLOYMENT.md step 6 (copy the seed file
in with `compose cp`, never pipe it).

## 9. Turn on continuous deployment

💻 GitHub → repo → Settings → Secrets and variables → Actions:

| Type | Name | Value |
|---|---|---|
| Secret | `DEPLOY_HOST` | the laptop's **tailnet** name, e.g. `mb-server` |
| Secret | `DEPLOY_USER` | `deploy` |
| Secret | `DEPLOY_SSH_KEY` | private half of a dedicated pair (`ssh-keygen -t ed25519 -f ci_deploy_key`); public half appended to 🖥️ `~/.ssh/authorized_keys` |
| Secret | `TS_OAUTH_CLIENT_ID` | from step 5.2 |
| Secret | `TS_OAUTH_SECRET` | from step 5.2 |
| Secret | `POSTGRES_PASSWORD` | same value as in `deploy/.env` |
| Secret | `GH_PAT` | the `read:packages` token |
| Variable | `TARGET_DIR` | `/opt/mountain-breath` |
| Variable | `DOMAIN` | `mountainbreath.com` |
| Variable | `DEPLOY_ENABLED` | `true` |

(The tunnel token stays only in the server's `deploy/.env` — CI never
needs it, since compose reads that file on the server.)

From then on: **merge to master → CI green → images published → the
runner joins the tailnet → the laptop pulls and restarts.** Verify with
a tiny change and `gh run watch`.

## 10. Backups

DEPLOYMENT.md step 7 unchanged (cron + `backup.sh`, test the restore
once). Home-specific addition: the laptop is a single point of failure
*in your house* — copy `/opt/backups` somewhere that isn't the same
machine now and then. Over the tailnet that's one command from 💻:

```powershell
scp deploy@mb-server:/opt/backups/mb_*.sql.gz D:\Backups\mountain-breath\
```

## Limits to know about

- **Uptime.** Power cut outlasting the battery, ISP outage, a borrowed
  ethernet cable — the shop is down and nobody pages you. Fine for a
  learning deployment; know it's the trade.
- **Upload bandwidth.** Residential upstream is thin. Cloudflare's edge
  caching of static assets absorbs much of it, but uncached requests
  (API, first hits) ride your line.
- **Cloudflare free plan.** Uploads cap at 100 MB per request (our API's
  own cap is 50 MB — never hit); ToS want normal website content, not a
  video CDN (our short product MP4s on product pages are fine).
- **TLS terminates at the edge** — Cloudflare can read the traffic. The
  #12 ideal, given up knowingly in #100 because CGNAT left no free
  alternative.

When the project outgrows the laptop, [DEPLOYMENT.md](DEPLOYMENT.md) is
the migration path: the same images on a VPS, where you either keep this
tunnel (it works anywhere) or reinstate Caddy + Let's Encrypt from the
repo's `deploy/Caddyfile` (kept for exactly that day) and get #12's
end-to-end TLS back. The domain moves with you either way.
