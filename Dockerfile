FROM node:24-alpine AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend-builder
WORKDIR /src
COPY go.mod go.sum ./
COPY third_party ./third_party
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/bloope ./cmd/server

FROM docker:cli AS docker-cli

FROM debian:bookworm-slim
WORKDIR /app

RUN for attempt in 1 2 3; do \
		apt-get update \
		&& apt-get install -y --no-install-recommends bash ca-certificates curl git openssh-client \
		&& rm -rf /var/lib/apt/lists/* \
		&& break; \
		if [ "$attempt" = "3" ]; then exit 1; fi; \
		sleep 5; \
	done \
	&& curl --retry 5 --retry-delay 3 -fsSL https://railpack.com/install.sh | sh

COPY --from=docker-cli /usr/local/bin/docker /usr/local/bin/docker
COPY --from=backend-builder /out/bloope /app/bloope
COPY --from=frontend-builder /src/frontend/dist /app/frontend/dist

ENV GIN_MODE=release \
	BLOOPE_FRONTEND_DIR=/app/frontend/dist \
	BLOOPE_DB_PATH=/app/data/bloope.db \
	BLOOPE_CADDYFILE_PATH=/app/tmp/caddy/Caddyfile \
	BLOOPE_CADDY_CONTAINER=bloope-compose-caddy \
	BLOOPE_CADDY_PORT=8081 \
	BLOOPE_DEPLOYMENT_HOST_SUFFIX=localhost \
	BLOOPE_PUBLIC_SCHEME=http \
	BLOOPE_CONTAINER_PORT=8080 \
	RAILPACK_BIN=/usr/local/bin/railpack \
	RAILPACK_USE_WSL=0 \
	BUILDKIT_HOST=docker-container://bloope-buildkit \
	PATH=/root/.local/bin:$PATH

EXPOSE 8080

CMD ["/app/bloope"]
