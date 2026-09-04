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
- User: any name — this laptop's is `capybara`; the VPS runbook's
  examples say `deploy`. Whatever you pick, the `DEPLOY_USER` secret in
  step 9 must name the same user, and every 🖥️ command in this runbook
  runs as that user.
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
     for the apex; `www` is a redirect, step 14 — never a second entry)
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
GRAFANA_PASSWORD=<another generated hex string — the Grafana admin login>
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
in with `compose cp`, never pipe it). One line tells you whether both
happened — a live shop wants a catalog and at least one admin:

```bash
docker compose -f deploy/docker-compose.prod.yml exec -T postgres psql -U mb -d mountain_breath -Atc \
  "select (select count(*) from products) || ' products, ' || (select count(*) from users where role='admin') || ' admins'"
# expect something like: 8 products, 1 admins
```

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

[DEPLOYMENT.md](DEPLOYMENT.md) step 7 applies unchanged — the systemd
timer, the restore drill, disaster recovery — and step 7b (the R2 copy)
is not optional here: the laptop is a single point of failure *in your
house* (theft, a power surge, a spilled tea), and a copy on the PC beside
it shares every one of those fates. R2 is in another building by
construction. Until 7b is configured, the manual escape hatch over the
tailnet is one command from 💻:

```powershell
scp deploy@mb-server:/opt/backups/mb_*.dump D:\Backups\mountain-breath\
```

## 11. Outgoing mail (decision #104)

Until this step, reset links and order confirmations land in the api log
(`docker compose … logs api`), because `MB_SMTP_ADDR` is unset. A home
connection cannot deliver mail itself: residential IP ranges sit on every
blocklist, most ISPs block outbound port 25, and sender reputation is
earned per address over months. So the api hands each message to a
**relay** that has that reputation — the same SMTP conversation it has
with Mailpit in dev, aimed at a different host, with a password.

**Concepts, briefly.** A receiving server asks three questions before it
trusts a message claiming to come from `mountainbreath.net`:

- **SPF** — a TXT record naming the servers allowed to send for the
  domain (the relay's, here).
- **DKIM** — the relay signs every message; a TXT record publishes the
  public key so receivers can check the signature. A signed package:
  anyone can verify, only the key holder can sign.
- **DMARC** — a TXT record saying what to do when both checks fail
  (`p=none` = deliver and report; tighten later once reports look clean).

Resend puts its SPF on a `send` subdomain and DKIM under
`resend._domainkey`, so nothing collides with the apex records Cloudflare
Email Routing adds for inbound mail.

**Provider:** Resend, over SMTP. Free tier: 3,000 messages a month, 100 a
day, no branding — Brevo's free plan stamps "Sent with Brevo" on every
message, which is not what a password reset should look like. The api's
one-method mailer stays exactly what dev exercises against Mailpit; only
the address and the credentials change.

💻 In the browser:

1. resend.com → sign up → **Domains → Add domain** `mountainbreath.net`.
   Resend shows three records; add them in Cloudflare → DNS, Proxy status
   **DNS only**, pasting names without the domain suffix: an MX on `send`
   (priority 10), a TXT SPF on `send`, a TXT DKIM on `resend._domainkey`.
   Click **Verify** — minutes, usually.
2. **API Keys → Create**: permission *Sending access*, restricted to the
   domain. Copy it now; it is shown once.
3. Cloudflare → **Email → Email Routing → Enable**, then a rule
   `hive@mountainbreath.net` → the family's real mailbox (confirm the
   destination when Cloudflare mails it). Replies to order mails now
   reach a human; the shop still sends nothing through Cloudflare, which
   only receives.
4. DMARC, once the first messages have gone out: Cloudflare → DNS → TXT
   `_dmarc` with `v=DMARC1; p=none; rua=mailto:hive@mountainbreath.net`.

🖥️ On the laptop:

```bash
cd /opt/mountain-breath/deploy
cat >> .env <<'EOF'
MB_SMTP_ADDR=smtp.resend.com:587
MB_SMTP_USERNAME=resend
MB_SMTP_PASSWORD=re_…the API key…
MB_MAIL_FROM=Mountain Breath <hive@mountainbreath.net>
EOF
cd .. && docker compose -f deploy/docker-compose.prod.yml up -d api   # recreates api with the new env
docker compose -f deploy/docker-compose.prod.yml logs api | grep "mail via SMTP"
```

**Test it the way a customer would:** 💻 on the live site's sign-in page,
follow *Forgot password* for your own account and watch the message
arrive (check the spam folder the first time; Resend's **Logs** page shows
the relay's side of every message for 30 days). A failure is a
`sending reset mail` line in the api log — the endpoint answers 204
either way, by design, so the caller learns nothing about which emails
exist.

Limits worth knowing: 100 messages a day on the free plan — a launch
week of resets and confirmations fits, a newsletter to hundreds of
subscribers does not (the §2 newsletter sender will need the paid tier
or batching). The api waits at most 10 seconds for the relay before
logging a failure and answering the customer anyway: a relay outage
slows a checkout, it never breaks one.

## 12. Alerts to your phone (decision #105)

Prometheus evaluates the rules in `deploy/observability/alerts.yml` (the
API down, a 5xx spike, slow requests, a saturated DB pool) and hands
firing alerts to Alertmanager; until this step Alertmanager only showed
them in its own UI, which nobody watches at 03:00. This wires a Telegram
receiver. A backup failure takes the same road: `backup.sh` fires a
`BackupFailed` alert through Alertmanager's API and resolves it on the
next success — one channel, one token holder, one place to silence
things during maintenance. `deploy/alert.sh` is that API call as a
script, for anything else that should page and for testing.

The two values live in two places on purpose. The **chat id** is not a
secret and goes in `deploy/.env`, where compose interpolates it into
Alertmanager's config (Alertmanager reads no environment variables
itself). The **bot token** is a secret and rides as a *file*, mounted by
compose's `secrets:` at `/run/secrets/telegram_token`. Both are
required, and both are checked in the same two places: the Alertmanager
container refuses to start without them and says which one is missing
in its log, and the CI deploy checks both before touching the stack.
Compose itself is deliberately *not* the check: it only warns about a
missing secret file and mounts an empty directory in its place, and a
hard `:?` on the chat id turned out to block every compose command on
the laptop, backups included, until the value existed.

💻 In Telegram:

1. Message **@BotFather** → `/newbot` → pick a name and a username ending
   in `bot`. It answers with the token (`123456:ABC-…`).
2. Open a chat with your new bot and send it anything — a bot cannot
   write to you first.
3. Get the chat id: open
   `https://api.telegram.org/bot<TOKEN>/getUpdates` in the browser and
   read `"chat":{"id":123456789,"first_name":"…","type":"private"}` from
   the reply — the `chat` object inside your message. Two traps: the
   number before the colon in the token is the *bot's* id, and using it
   ends in `the bot can't send messages to the bot (403)`; and an empty
   `"result":[]` means the bot has no message from you yet (send one and
   reload). For a family group: add the bot to the group, post once,
   same URL — group ids are negative.

🖥️ On the laptop:

```bash
cd /opt/mountain-breath/deploy
printf '%s' '123456:ABC-the-token' > observability/telegram.token   # git-ignored via *.token
# The image runs Alertmanager as `nobody` (uid 65534), and a bind-mounted
# file keeps its host owner and mode: a 600 file owned by you exists and
# is non-empty inside the container, and is still unreadable there. Hand
# it to that uid, read-only. (Replacing the token later: `sudo tee`.)
sudo chown 65534:65534 observability/telegram.token
sudo chmod 400 observability/telegram.token
echo 'TELEGRAM_CHAT_ID=123456789' >> .env
# --force-recreate, not a plain up: compose renders the inline config
# when it CREATES the container and does not count that text in its
# "did the service change?" hash — a corrected chat id would otherwise be
# ignored with a cheerful "Running".
cd .. && docker compose -f deploy/docker-compose.prod.yml up -d --force-recreate alertmanager
docker compose -f deploy/docker-compose.prod.yml exec -T alertmanager grep chat_id /etc/alertmanager/alertmanager.yml   # your id
docker compose -f deploy/docker-compose.prod.yml logs alertmanager | tail -5   # no "error" lines

# Test the whole path without an outage: fire a synthetic alert, WAIT
# for the phone to buzz, then send the all-clear. The wait is not
# optional: Alertmanager holds a new alert for group_wait (30 s) before
# its first message, and an alert resolved inside that window sends
# nothing at all — not even "resolved", since nothing was ever sent as
# firing.
bash deploy/alert.sh fire TestAlert "hello from the laptop"
sleep 45                                   # phone buzzes during this
bash deploy/alert.sh resolve TestAlert     # a second message: resolved
docker compose -f deploy/docker-compose.prod.yml logs --tail 20 alertmanager   # no "Notify attempt failed" = healthy
```

Deploy ordering: the compose change that carries this arrives with the
next master deploy, whose script checks for the token file and the chat
id before touching the stack — a missing one fails the
job with a readable message and leaves the running containers as they
were. Do this step before merging, or expect one red deploy that turns
green on re-run.

What pages, and when:

| Alert | Fires when | Source |
|---|---|---|
| APIDown | Prometheus cannot scrape the API for 1 min | alerts.yml |
| HighErrorRate | more than 0.5 server errors/s for 5 min | alerts.yml |
| SlowRequests | p95 latency above 500 ms for 10 min | alerts.yml |
| DBPoolSaturated | handlers wait over 100 ms/s for pool connections, 5 min | alerts.yml |
| BackupFailed | the nightly backup exits non-zero; resolves on the next success | backup.sh |

A firing alert repeats every 4 h until it resolves. Silence one during
planned work from Alertmanager's UI over the tailnet (Observability,
below).

## 13. Google sign-in in production (E8's button, decision #5)

The code has been ready since E8: with `MB_GOOGLE_CLIENT_ID` and
`MB_GOOGLE_CLIENT_SECRET` set, *Continue with Google* works; unset, the
button explains itself. What production needs is a client that trusts
the real callback URL, and a consent screen the public may use.

💻 console.cloud.google.com → the project from E8 (or a new one):

1. **APIs & Services → Credentials** → the OAuth client (type *Web
   application*) → **Authorized redirect URIs** → add
   `https://mountainbreath.net/api/v1/auth/oauth/google/callback`
   (keep the localhost one for dev). Save; copy the Client ID and the
   Client secret.
2. **OAuth consent screen** (newer consoles: *Google Auth Platform →
   Audience*): move from **Testing** to **In production** (*Publish
   app*). Testing mode caps sign-in at 100 listed test users and expires
   their tokens after a week. The basic scopes this app asks for (email,
   profile) need no verification review, so publishing is one click.
   Branding: app name "Mountain Breath", a support email, the home page.

🖥️ On the laptop:

```bash
cd /opt/mountain-breath/deploy
cat >> .env <<'EOF'
MB_GOOGLE_CLIENT_ID=…apps.googleusercontent.com
MB_GOOGLE_CLIENT_SECRET=GOCSPX-…
EOF
cd .. && docker compose -f deploy/docker-compose.prod.yml up -d api
```

**Test:** 💻 sign out of the live site, click *Continue with Google*,
pick an account that has no Mountain Breath password — you land signed
in, and the account page shows the Google-linked identity. A
`redirect_uri_mismatch` page from Google means step 1's URI differs from
`MB_PUBLIC_URL` + the callback path, character for character (scheme,
host, no trailing slash).

## 14. `www.mountainbreath.net` → the apex

Not a second tunnel hostname. The site's `rel=canonical` and `hreflang`
tags are built from `window.location.origin`, so serving the same pages
at two hosts would publish two canonical URLs for every page — the
duplicate-content problem E10's tags exist to prevent. The answer is
one hostname and a redirect for the other, answered at Cloudflare's
edge without ever reaching the laptop.

💻 Cloudflare dashboard → the domain:

1. **DNS → Add record**: type `CNAME`, name `www`, target
   `mountainbreath.net`, Proxy status **Proxied** (orange cloud). The
   record only has to exist so Cloudflare answers for the name; the
   proxy is what lets a rule intercept it.
2. **Rules → Redirect Rules → Create rule** → template **Redirect from
   WWW to Root**. The template opens the **wildcard pattern** form;
   fill it in explicitly rather than keeping the prefilled
   `https://www.*`:

   | Field | Value |
   |---|---|
   | Request URL | `https://www.mountainbreath.net/*` |
   | Target URL | `https://mountainbreath.net/${1}` |
   | Status code | `301 - Permanent Redirect` |
   | Preserve query string | **checked** |

   `${1}` is whatever the `*` captured, so the path survives — without
   it every deep link lands on the home page. The wildcard matches the
   path only, which is why the query string needs its own checkbox:
   unchecked, `?lang=hy` and every UTM parameter on a shared link is
   dropped at the redirect. Deploy.

   (Equivalent by hand, and scheme-independent, if you prefer the
   **custom filter expression** radio: `http.host eq
   "www.mountainbreath.net"` → dynamic redirect to
   `concat("https://mountainbreath.net", http.request.uri.path)`.)

3. **SSL/TLS → Edge Certificates → Always Use HTTPS: On.** The rule
   above matches URLs beginning `https://`, so without this a request
   to `http://www.…` never matches it and is served as a second
   hostname — precisely the duplicate content the rule exists to
   prevent. With it, http first 301s to https, then the rule fires.

   That toggle warns about `ERR_TOO_MANY_REDIRECTS` "if your origin
   also forces HTTPS redirects". Ours does not, and the proof is one
   line of [nginx.conf](../frontend/nginx.conf): `listen 80;` with no
   `return 301 https://` anywhere, and no scheme redirect in the API
   either. The loop it warns about needs an origin that answers "use
   https" to the plain-http request Cloudflare forwards; ours never
   says that.

   The **SSL/TLS encryption mode** (Full, Full (strict), …) is not a
   gap here and the newer dashboard may not even show the picker —
   that setting governs how Cloudflare *dials* an origin, and with a
   named tunnel there is no inbound origin connection to dial:
   `cloudflared` opens the connection outward and is authenticated by
   its token. Leave whatever is there.

**Test** from anywhere — both schemes, with a query string, because
those are the two things that quietly break:

```powershell
curl.exe -sIL "http://www.mountainbreath.net/shop?lang=hy" | Select-String "HTTP/|location"
# HTTP/1.1 301 Moved Permanently          ← Always Use HTTPS
# location: https://www.mountainbreath.net/shop?lang=hy
# HTTP/1.1 301 Moved Permanently          ← the redirect rule
# location: https://mountainbreath.net/shop?lang=hy
# HTTP/1.1 200 OK                         ← the apex, path and query intact
```

Two hops for a plain-http visitor is expected and both are answered at
the edge. A `200` on the *first* line instead of a `301` means the www
CNAME is grey-clouded: Cloudflare is answering DNS but not proxying,
so no rule ever sees the request.

## Observability (decision #101)

The prod stack carries the same Prometheus + Alertmanager + Grafana trio
as the local one, with one deliberate difference: **every port is bound
to `127.0.0.1`** — answerable only from the laptop itself. Grafana is a
password-protected admin surface (bots scan for those) and Prometheus
has no auth at all, so neither is ever added to the tunnel; the shop is
public, the graphs are private.

You reach them over the tailnet from anywhere — the SSH `-L` flag makes
your browser's localhost port travel to the laptop's:

```powershell
# The LEFT port is on your own machine and is a free choice — 3300/9390
# here because the local dev stack usually already occupies 3000/9090
# (forwarding onto a taken port either fails to bind or, worse, shows
# you the LOCAL Grafana while you believe you're looking at prod).
ssh -L 3300:localhost:3000 -L 9390:localhost:9090 capybara@homeserver
# then in the browser, while that ssh stays open:
#   http://localhost:3300  → prod Grafana (admin / GRAFANA_PASSWORD from deploy/.env)
#   http://localhost:9390  → prod Prometheus's query UI
```

The dashboards and alert rules are provisioned from
`deploy/observability/` (the CI deploy copies that directory alongside
the compose file). Alerts reach the family's phone through the Telegram
receiver of step 12; this UI is where one gets silenced during planned
work.

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
