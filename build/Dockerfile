FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/rhmi-operator \
    USER_UID=1001 \
    USER_NAME=integreatly-operator \
    TEMPLATE_PATH=/usr/local/bin/templates/monitoring \
    MANIFEST_DIR=/usr/local/bin/manifests \
    PRODUCT_DECLARATION=/usr/local/bin/products/installation.yaml

# install operator binary
COPY build/_output/bin/integreatly-operator ${OPERATOR}
# copy templates
COPY templates /usr/local/bin/templates

COPY products /usr/local/bin/products

COPY manifests ${MANIFEST_DIR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
