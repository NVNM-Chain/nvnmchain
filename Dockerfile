# syntax=docker/dockerfile:1

ARG GO_VERSION="1.23"
ARG IMG_TAG="latest"
ARG BUILD_TAGS="netgo,ledger,muslc"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM golang:${GO_VERSION}-alpine3.20 AS mantra-builder
WORKDIR /src/app/
RUN apk add --no-cache \
    ca-certificates \
    build-base \
    linux-headers \
    binutils-gold \
    git

# Download go dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/nonroot/.cache/go-build \
    --mount=type=cache,target=/nonroot/go/pkg/mod \
    go mod download

# Copy the remaining files
COPY . .

# Build mantrachaind binary
# build tag info: https://github.com/cosmos/wasmd/blob/master/README.md#supported-systems
RUN --mount=type=cache,target=/nonroot/.cache/go-build \
    --mount=type=cache,target=/nonroot/go/pkg/mod \
    LEDGER_ENABLED=true BUILD_TAGS='muslc osusergo' LINK_STATICALLY=true make build

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM alpine:$IMG_TAG
RUN apk add --no-cache build-base jq
RUN addgroup -g 1025 nonroot
RUN adduser -D nonroot -u 1025 -G nonroot
ARG IMG_TAG
COPY --from=mantra-builder /src/app/build/mantrachaind /usr/local/bin/
EXPOSE 26656 26657 1317 9090
USER nonroot

ENTRYPOINT ["mantrachaind", "start"]