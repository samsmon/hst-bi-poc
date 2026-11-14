# Single image: builds the Svelte app, embeds it into the Go binary.
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.27-bookworm AS build
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=web /src/web/dist/ ./internal/webui/dist/
# cgo is required by pg_query_go (libpg_query).
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o /out/hst-bi ./cmd/hst-bi

FROM gcr.io/distroless/cc-debian12:nonroot
COPY --from=build /out/hst-bi /hst-bi
ENV BI_ADDR=0.0.0.0:8610
EXPOSE 8610
ENTRYPOINT ["/hst-bi"]
