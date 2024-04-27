# First stage: build the Go binary
FROM --platform=$BUILDPLATFORM docker.io/golang:1.22.1 AS go-builder

WORKDIR /app

RUN \
  --mount=type=bind,source=./go.mod,target=/app/go.mod \
  --mount=type=bind,source=./go.sum,target=/app/go.sum \
  go mod download

COPY . .

ARG TARGETARCH
RUN \
  GOOS=linux \
  GOARCH=$TARGETARCH \
  CGO_ENABLED=0 \
  go build -a -installsuffix cgo -o ./app .

FROM scratch

EXPOSE 8080

LABEL maintainer="kin.wai.koo@gmail.com"

# Copy the Go binary from the first stage
COPY --from=go-builder --chown=1000:0 --chmod=775 /app/app /usr/local/bin/app

CMD ["/usr/local/bin/app"]
