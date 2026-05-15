# Copyright (c) 2025, 2026 IBM Corp.
# SPDX-License-Identifier: Apache-2.0

ARG BASE_UBI_IMAGE_TAG=9.6
ARG BUILDER_IMAGE
FROM ${BUILDER_IMAGE:-registry.access.redhat.com/ubi9/go-toolset:9.6-1754467841} AS builder
ARG TARGETOS
ARG TARGETARCH
USER root

WORKDIR /workspace

# build pod validator
COPY go.mod .
COPY go.sum .
COPY vendor/ vendor/
COPY pkg/ pkg/

COPY main.go main.go

ARG BUILD_FLAGS=""

ARG GOLANG_VERSION
ENV GOTOOLCHAIN="go${GOLANG_VERSION}"

RUN echo "TARGETARCH = '${TARGETARCH}' TARGETOS='${TARGETOS}'" && \
    echo "GO ENV DUMP: " && go env GOVERSION && go env GOTOOLDIR && \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on \
    go build ${BUILD_FLAGS} -mod vendor -tags strictfipsruntime -a -o spyre-webhook-validator main.go

# install useradd command and upgrade libcap
RUN dnf --installroot=/tmp/ubi-micro \
    --nodocs --setopt=install_weak_deps=False \
    install -y \
    shadow-utils openssl-libs openssl-fips-provider libcap-2.48-10.el9_7.1 && \
    dnf --installroot=/tmp/ubi-micro clean all

# generate minimal image
FROM registry.access.redhat.com/ubi9/ubi-micro:${BASE_UBI_IMAGE_TAG}

WORKDIR /
ARG VERSION

LABEL io.k8s.display-name="IBM Spyre Operator Validator Webhook"
LABEL name="IBM Spyre Operator Validator Webhook"
LABEL vendor="IBM"
LABEL version="${VERSION}"
LABEL release="N/A"
LABEL summary="Automate the management and monitoring of IBM Spyre devices."
LABEL description="See summary"

COPY --from=builder /tmp/ubi-micro/ /
COPY ./LICENSE /licenses/LICENSE
RUN useradd spyre-pod-validator
USER spyre-pod-validator

# Copy the binary from the builder stage
COPY --from=builder /workspace/spyre-webhook-validator /usr/local/bin/spyre-webhook-validator

EXPOSE 8443 8081 8080

HEALTHCHECK NONE

# Command to run the application
CMD ["spyre-webhook-validator"]
