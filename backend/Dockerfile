# ---- build stage: full Go toolchain (~800MB, discarded after build) ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Copy the dependency manifests FIRST and download modules as a separate
# layer: as long as go.mod/go.sum don't change, rebuilds skip the download.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 → fully static binary, runnable on a bare image.
# -s -w strips debug info: smaller binary (we debug with source locally).
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api
# Pre-create the uploads dir here (the final image has no shell to mkdir).
RUN mkdir -p /out/uploads

# ---- final stage: distroless = no shell, no package manager, no OS junk ----
# Tiny attack surface and a ~15MB image. Runs as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
# Owned by nonroot (uid 65532): when Docker first initializes the named
# volume mounted here, it copies this directory's OWNERSHIP into the volume —
# without this, the volume belongs to root and the nonroot api can't write.
COPY --from=build --chown=65532:65532 /out/uploads /uploads
EXPOSE 8080
ENTRYPOINT ["/api"]
