# Execute this docker file via make bundle/build command with VERSION env

FROM scratch

ARG version
ARG manifest_path=bundles/managed-api-service/${version}/manifests
ARG metadata_path=bundles/managed-api-service/${version}/metadata

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=managed-api-service
LABEL operators.operatorframework.io.bundle.channels.v1=stable
LABEL operators.operatorframework.io.bundle.channel.default.v1=stable

COPY ${manifest_path} /manifests/
COPY ${metadata_path} /metadata/
