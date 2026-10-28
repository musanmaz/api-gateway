FROM golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/control-api ./cmd/control-api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/agent ./cmd/agent \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/backend

FROM debian:bookworm-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8 AS runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN useradd --uid 65532 --create-home --user-group nonroot

FROM runtime AS api
COPY --from=build /out/control-api /usr/local/bin/control-api
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/control-api"]

FROM runtime AS agent
COPY --from=build /out/agent /usr/local/bin/agent
COPY deploy/docker/agent-entrypoint.sh /usr/local/bin/agent-entrypoint.sh
RUN chmod 755 /usr/local/bin/agent-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/agent-entrypoint.sh"]

FROM runtime AS backend
COPY --from=build /out/backend /usr/local/bin/backend
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/backend"]
