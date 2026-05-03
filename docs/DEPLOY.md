# Hosting & Deployment

The app is a single Go binary plus a static folder, plus the
`pdftotext` and `pdftoppm` system tools. Three deployment paths in
increasing order of effort:

## Option 1 — Fly.io (recommended for a side project)

Cheapest path that scales. Free tier covers small traffic.

```bash
# install flyctl
curl -L https://fly.io/install.sh | sh

# from the project root
fly launch --no-deploy        # creates fly.toml
fly secrets set SITE_URL=https://yourdomain.com SITE_HOST=yourdomain.com
fly deploy
```

Use this `Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/pdfrep .

FROM alpine:3.20
RUN apk add --no-cache poppler-utils ca-certificates
WORKDIR /app
COPY --from=build /out/pdfrep /app/pdfrep
COPY web        /app/web
COPY texts      /app/texts
COPY colors     /app/colors
COPY public     /app/public
EXPOSE 8080
ENV ADDR=:8080
CMD ["/app/pdfrep"]
```

Point your domain at the Fly app:

```bash
fly certs create yourdomain.com
fly certs create www.yourdomain.com
```

…then add the DNS records Fly shows.

## Option 2 — VPS (Hetzner / DigitalOcean / Vultr)

A €4/mo Hetzner box is plenty for tens of thousands of edits a day.

1. Spin up a small Linux box (Ubuntu 24.04).
2. `apt install poppler-utils nginx certbot python3-certbot-nginx`
3. Build the binary on your laptop and `scp` it up, or build on the
   server with `apt install golang-1.22 && go build`.
4. Run via systemd. Drop this in `/etc/systemd/system/pdfrep.service`:

   ```ini
   [Unit]
   Description=PDF Text Replace
   After=network.target

   [Service]
   User=www-data
   WorkingDirectory=/opt/pdfrep
   ExecStart=/opt/pdfrep/pdfrep -addr 127.0.0.1:8080
   Environment=SITE_URL=https://yourdomain.com SITE_HOST=yourdomain.com
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```

5. Front it with nginx + Let's Encrypt:

   ```nginx
   server {
     listen 80;
     server_name yourdomain.com www.yourdomain.com;
     client_max_body_size 30M;
     location / { proxy_pass http://127.0.0.1:8080; proxy_set_header Host $host; }
   }
   ```

   Then `certbot --nginx -d yourdomain.com -d www.yourdomain.com`.

## Option 3 — Cloud Run (Google) / Render / Railway

All work. Cloud Run is great because it scales to zero. Same Dockerfile
as Fly. The only gotcha: Cloud Run's 32 MB request size limit means you
must keep the upload limit under that.

## Always

- Put it behind **Cloudflare** for free DDoS protection, caching, and
  global edge.
- Set `SITE_URL` and `SITE_HOST` env vars so canonical URLs and sitemap
  point at the right domain.
- Add a cron / k6 job to clean `storage/uploads/*` and `storage/output/*`
  older than 1 hour. Sample one-liner:

  ```bash
  find storage -mindepth 1 -maxdepth 2 -type d -mmin +60 -exec rm -rf {} +
  ```

- Set up an uptime monitor (UptimeRobot has a free tier).
