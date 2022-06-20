# Build the manager binary
FROM registry.ci.openshift.org/openshift/release:golang-1.18 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY version/ version/
COPY test/ test/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o rhoam-operator main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/rhoam-operator \
    USER_UID=1001 \
    USER_NAME=integreatly-operator

COPY --from=builder /workspace/rhoam-operator /usr/local/bin/rhoam-operator

COPY templates /templates

COPY manifests /manifests

COPY products /products

COPY build/bin /usr/local/bin
RUN /usr/local/bin/user_setup


ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
