# Deployment Runbook

From an empty VPS to a live, HTTPS-served, auto-deploying shop. Written for
Ubuntu 24.04 LTS on any provider (Hetzner/DigitalOcean/etc., smallest tier
is plenty). Commands marked 💻 run on your Windows machine, 🖥️ on the server.

> **The deployment actually in use is a home laptop + Cloudflare named
> tunnel** — [DEPLOYMENT_HOME.md](DEPLOYMENT_HOME.md) (decision #100,
> reversing #12 after the ISP turned out to run CGNAT). This file stays as
> the VPS variant and migration target. Note for that day:
> `docker-compose.prod.yml` is now tunnel-shaped (cloudflared, no Caddy);
> on a VPS you either keep the tunnel or reinstate a Caddy service using
> the kept `deploy/Caddyfile` — see the last section of the home runbook.

## Interim: public demo URL with no VPS and no domain

Until the VPS exists, a **Cloudflare quick tunnel** exposes the local
containerized stack at a free `https://<random>.trycloudflare.com` URL —
no account, no DNS, no open ports (cloudflared connects *outbound* to
Cloudflare; visitors are relayed back down that connection). 💻:

```powershell
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.tunnel.yml up -d --build
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.tunnel.yml logs cloudflared   # the URL is in here
```

Seed + admin promotion are the same as step 6 below (swap in
`docker-compose.yml`). Limits to know about:

- The URL is **random and changes every restart** of the cloudflared
  container — re-share it each time. A stable URL needs a domain in
  Cloudflare (named tunnel), which arrives with the VPS anyway.
- The site is up only while this machine is on; no uptime promise.
- Google sign-in stays off (its redirect URI must be registered per exact
  origin — pointless with a churning URL). Leave the keys unset and the
  button explains itself.
- Password-reset emails would link to `MB_PUBLIC_URL`; with SMTP unset they
  land in the api log anyway, so leave both alone for demos.
- Keep `MB_ENV=dev`: the browser↔Cloudflare leg is HTTPS, but the
  cloudflared→nginx leg is plain HTTP on the compose network, which is fine
  for a demo and required until real TLS termination exists end-to-end.

## 0. What you need first

- A VPS (2GB RAM is comfortable) — note its public IP
- A domain (or subdomain) you control
- ~1 hour

## 1. DNS

At your domain registrar, create an **A record** pointing to the server IP:

```
shop.example.com  A  <server-ip>      (TTL 300 while setting up)
```

Do this FIRST — Let's Encrypt needs the name to resolve before it issues
a certificate. Verify: 💻 `nslookup shop.example.com`.

## 2. First login & hardening

💻 `ssh root@<server-ip>` (password/key from your provider), then 🖥️:

```bash
# a working user instead of root
adduser deploy
usermod -aG sudo deploy

# your SSH key for that user (paste your PUBLIC key)
mkdir -p /home/deploy/.ssh
nano /home/deploy/.ssh/authorized_keys      # paste key, save
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh && chmod 600 /home/deploy/.ssh/authorized_keys

# lock the door: no root login, no passwords over SSH
sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
systemctl restart ssh

# firewall: only SSH + web
apt update && apt install -y ufw fail2ban unattended-upgrades
ufw allow OpenSSH && ufw allow 80/tcp && ufw allow 443/tcp
ufw --force enable
```

⚠️ Before closing this session, verify from a NEW terminal that
💻 `ssh deploy@<server-ip>` works — never lock yourself out.

(No Windows key yet? 💻 `ssh-keygen -t ed25519` → key lands in
`~/.ssh/id_ed25519.pub`.)

## 3. Install Docker

🖥️ as `deploy`:

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker deploy
# log out and back in, then verify:
docker ps
```

## 4. Give the server read access to the repo and images

The repo itself is public (clone needs no credential), but the GHCR images
are private — so the server needs the pull token in (b). The deploy key in
(a) is only required if the repo is ever made private again; skip it
otherwise and clone over HTTPS.

**a) Deploy key (git clone/pull) — optional while the repo is public.**
🖥️ `ssh-keygen -t ed25519 -f ~/.ssh/repo_key -N ""`,
then 💻 add the PUBLIC key at GitHub → repo → Settings → Deploy keys
(read-only). 🖥️ Tell git to use it:

```bash
cat >> ~/.ssh/config <<'EOF'
Host github.com
  IdentityFile ~/.ssh/repo_key
EOF
```

**b) GHCR pull token.** 💻 GitHub → Settings → Developer settings →
Personal access tokens (classic) → generate with ONLY `read:packages`.
🖥️ `docker login ghcr.io -u Nerses01` (paste the token as password).

## 5. Clone and configure

🖥️:

```bash
sudo mkdir -p /opt/mountain-breath && sudo chown deploy:deploy /opt/mountain-breath
git clone git@github.com:Nerses01/Mountain_Breath.git /opt/mountain-breath
cd /opt/mountain-breath/deploy

# production secrets — NEVER the dev password
cat > .env <<'EOF'
POSTGRES_USER=mb
POSTGRES_PASSWORD=<generate a long random one>
POSTGRES_DB=mountain_breath
DOMAIN=shop.example.com
EOF
chmod 600 .env
```

## 6. First boot

🖥️ from `/opt/mountain-breath`:

```bash
docker compose -f deploy/docker-compose.prod.yml pull
docker compose -f deploy/docker-compose.prod.yml up -d
docker compose -f deploy/docker-compose.prod.yml ps   # wait: everything healthy
```

Open `https://shop.example.com` — certificate and redirect are automatic
(Caddy). Then seed the catalog and create your admin:

```bash
# COPY the file in; do not pipe it. Piping runs the stream through the shell's
# encoding — on a Windows host that silently mangles every non-ASCII byte, so
# the Armenian and Russian translations land as mojibake or as literal '?'.
# Nothing errors: the result is valid UTF-8, just wrong. `cp` moves raw bytes.
docker compose -f deploy/docker-compose.prod.yml cp backend/seed/seed.sql postgres:/tmp/seed.sql
docker compose -f deploy/docker-compose.prod.yml exec -T postgres \
  psql -U mb -d mountain_breath -v ON_ERROR_STOP=1 -f /tmp/seed.sql
# register your account through the website UI first, then:
docker compose -f deploy/docker-compose.prod.yml exec postgres \
  psql -U mb -d mountain_breath -c "UPDATE users SET role='admin' WHERE email='you@example.com';"
```

## 7. Backups

The shop keeps state in two places, so a backup has two parts:

| State | Lives in | Backed up as |
|---|---|---|
| Catalog, users, orders, sessions — everything relational | Postgres (`pgdata_prod` volume) | `mb_<stamp>.dump`, a `pg_dump --format=custom` archive taken inside the container |
| Product photos and videos | the `uploads_data` volume | `uploads_<stamp>.tar.gz`, written only when the volume's contents changed |

A dump without the files restores a catalog of broken images; the files
without the dump are anonymous JPEGs. `deploy/backup.sh` writes both to
`/opt/backups`, verifies the dump parses, keeps the newest 14 of each,
and — once 7b is configured — copies them off the machine.

🖥️ Install the schedule. A systemd timer rather than cron, for one
property: `Persistent=true` runs a missed 03:30 backup after the machine
was off, where cron would skip the night.

```bash
sudo install -d -o deploy -g deploy /opt/backups
sudo cp /opt/mountain-breath/deploy/systemd/mb-backup.{service,timer} /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now mb-backup.timer
systemctl list-timers mb-backup.timer        # NEXT / LAST columns
```

**Run one now, then prove it restores** — an untested backup is a hope,
not a backup. The drill restores the dump into a scratch database
(`mb_restore_drill`) inside the running Postgres, compares row counts
with the live database, reads the uploads archive back, and drops the
scratch database whatever happens:

```bash
cd /opt/mountain-breath
bash deploy/backup.sh
bash deploy/restore.sh --drill latest        # ends with "DRILL PASSED"
journalctl -u mb-backup --since yesterday    # what the timer's runs said
```

Repeat the drill after any Postgres major-version change and once a
quarter regardless: the point is that the restore path gets exercised by
someone who is not panicking. The same two commands run on a dev machine
against the dev stack (Git Bash included), which is how the scripts were
first proven:

```bash
export MB_BACKUP_DIR=/tmp/mb-backups \
       MB_COMPOSE_FILE=deploy/docker-compose.dev.yml \
       MB_UPLOADS_VOLUME=<any uploads volume; it is mounted read-only> \
       MB_ALERTMANAGER_URL=        # empty: a rehearsal pages nobody
bash deploy/backup.sh && bash deploy/restore.sh --drill latest
```

**Disaster recovery** — the only time `--real` is typed. It stops the
api, drops and recreates the database from the dump, replaces the
uploads volume, and starts the stack; the migrate job then carries the
schema forward if the images are newer than the archive:

```bash
bash deploy/restore.sh --real /opt/backups/mb_<stamp>.dump   # asks you to type the database name
```

### 7b. Off-machine copies (Cloudflare R2)

Two copies on one disk are one copy. R2 is S3-compatible object storage
on the Cloudflare account the tunnel already uses; the free tier (10 GB,
no egress fees) holds years of this shop's dumps. `rclone` does the
copying — it speaks S3 and dozens of other backends, so the remote can
change later without touching the script.

💻 Cloudflare dashboard → **R2 Object Storage**:

1. Create bucket `mountain-breath-backups` (any location hint).
2. **Manage R2 API Tokens → Create**: permission *Object Read & Write*,
   scoped to that one bucket. Note the **Access Key ID**, the **Secret
   Access Key**, and the account's S3 endpoint shown on the same page
   (`https://<account-id>.r2.cloudflarestorage.com`).

🖥️ On the server, as `deploy`:

```bash
sudo apt install -y rclone
mkdir -p ~/.config/rclone && chmod 700 ~/.config/rclone
cat > ~/.config/rclone/rclone.conf <<'EOF'
[r2]
type = s3
provider = Cloudflare
access_key_id = <Access Key ID>
secret_access_key = <Secret Access Key>
endpoint = https://<account-id>.r2.cloudflarestorage.com
acl = private
# a bucket-scoped token may neither list nor create buckets — don't try
no_check_bucket = true
EOF
chmod 600 ~/.config/rclone/rclone.conf
rclone ls r2:mountain-breath-backups        # empty listing, no error = credentials work

cp deploy/backup.env.example deploy/backup.env   # then uncomment MB_BACKUP_REMOTE
bash deploy/backup.sh && rclone ls r2:mountain-breath-backups
```

Retention on the remote: dumps older than `MB_REMOTE_KEEP_DAYS` (60) are
deleted by the script — only after a successful upload in the same run,
never by a bucket lifecycle rule (the script's comment on age-based
pruning says why). Uploads archives are kept: each is a distinct state
of the volume, and they only appear when something changed.

Restoring on a new machine starts by pulling the files back —
`rclone copy r2:mountain-breath-backups /opt/backups` — then `--real` as
above.

## 8. Turn on continuous deployment

💻 GitHub → repo → Settings → Secrets and variables → Actions:

| Type | Name | Value |
|---|---|---|
| Secret | `DEPLOY_HOST` | server IP |
| Secret | `DEPLOY_USER` | `deploy` |
| Secret | `DEPLOY_SSH_KEY` | a PRIVATE key whose public half is in the server's `authorized_keys` (generate a dedicated pair: `ssh-keygen -t ed25519 -f ci_deploy_key`) |
| Variable | `DEPLOY_ENABLED` | `true` |

From then on: **merge to master → CI green → images published → server
pulls and restarts.** Verify by pushing a tiny change and watching
`gh run watch`, then refreshing the site.

## 9. Rollback

Deploy a known-good build by SHA (from the GitHub commit list):

```bash
cd /opt/mountain-breath
# temporarily pin images in deploy/.env-style override or:
docker compose -f deploy/docker-compose.prod.yml up -d \
  --no-deps api web   # after editing the image tags to :<sha>
```

(Any green master commit has `:<sha>`-tagged images in GHCR.)

## Routine operations

| Task | Command (🖥️ from /opt/mountain-breath) |
|---|---|
| See status | `docker compose -f deploy/docker-compose.prod.yml ps` |
| Tail API logs | `docker compose -f deploy/docker-compose.prod.yml logs -f api` |
| Manual deploy | `git pull && docker compose -f deploy/docker-compose.prod.yml pull && docker compose -f deploy/docker-compose.prod.yml up -d` |
| Backup now | `bash deploy/backup.sh` (the timer does this nightly at 03:30) |
| Backup status | `systemctl list-timers mb-backup.timer` · `journalctl -u mb-backup --since yesterday` |
| Prove a backup restores | `bash deploy/restore.sh --drill latest` |
| Restore for real (disaster) | `bash deploy/restore.sh --real /opt/backups/mb_<stamp>.dump` — step 7 |
