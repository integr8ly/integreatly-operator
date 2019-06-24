FROM registry.access.redhat.com/ubi8/ubi:8.0

ENV OPERATOR=/usr/local/bin/integreatly-operator \
    USER_UID=1001 \
    USER_NAME=integreatly-operator

# install operator binary
COPY build/_output/bin/integreatly-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
