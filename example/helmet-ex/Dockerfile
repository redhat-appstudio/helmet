#
# Build
#

FROM registry.access.redhat.com/ubi9/go-toolset:latest AS builder

# Build arguments forwarded to "make build" as variable overrides.
ARG BUILD_VERSION=v0.0.0-SNAPSHOT
ARG COMMIT_ID=unknown

# Allows Go to automatically download the toolchain version required by "go.mod".
ENV GOTOOLCHAIN=auto

USER root

# Source and vendored dependencies (filtered by ".dockerignore").
WORKDIR /workspace
COPY . .

RUN make build VERSION="${BUILD_VERSION}" COMMIT_ID="${COMMIT_ID}"

#
# Run
#

FROM registry.access.redhat.com/ubi9-micro:latest

COPY --from=builder \
    /workspace/example/helmet-ex/helmet-ex \
    /usr/local/bin/helmet-ex

ENTRYPOINT ["helmet-ex"]
