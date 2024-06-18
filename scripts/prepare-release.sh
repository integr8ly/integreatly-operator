#!/usr/bin/env bash
set -e
set -o pipefail

if [[ -z "$OLM_TYPE" ]]; then
  OLM_TYPE="managed-api-service"
fi

case $OLM_TYPE in
"managed-api-service")
  PREVIOUS_VERSION=$(grep $OLM_TYPE bundles/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $3}') || echo "No previous version"
  PACKAGE_NAME=managed-api-service
  OPERATOR_TYPE=rhoam
  ;;
"multitenant-managed-api-service")
  PREVIOUS_VERSION=$(grep managed-api-service bundles/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $3}') || echo "No previous version"
  PACKAGE_NAME=managed-api-service
  OPERATOR_TYPE=multitenant-rhoam
  ;;
*)
  echo "Invalid OLM_TYPE set"
  echo "Use \"managed-api-service\" or \"multitenant-managed-api-service\""
  exit 1
  ;;
esac

if [[ -z "$ORG" ]]; then
  ORG="integreatly"
else
  ORG="$ORG"
fi

# Optional environment variable to set a different Kustomize path. If this
# variable is not set, it will use the one from the $PATH or try a default Kustomize path
if [[ -z $KUSTOMIZE_PATH ]]; then
  KUSTOMIZE="/usr/local/bin/kustomize"
else
  KUSTOMIZE=$(which kustomize)
fi

echo "Using kustomize path: $KUSTOMIZE"

# Path to gofmt
if [[ -z $GOROOT ]]; then
  GOFMT="/usr/bin/gofmt"
else
  GOFMT="$GOROOT/bin/gofmt"
fi

# The base CSV is used to generate the final CSV by combining it with the other operator
# manifests. In operator-sdk v1.2.0, the replaces field of the new CSV is set from
# the current version of **the base CSV**, so we need to update the base CSV in order
# for the replaces field to be set when generating the next release
update_base_csv() {
  yq e -i ".metadata.name=\"managed-api-service.v$VERSION\"" config/manifests-$OPERATOR_TYPE/bases/managed-api-service.clusterserviceversion.yaml
  yq e -i ".spec.version=\"$VERSION\"" config/manifests-$OPERATOR_TYPE/bases/managed-api-service.clusterserviceversion.yaml
  if [[ "${VERSION}" != "${PREVIOUS_VERSION}" ]]; then
    yq e -i ".spec.replaces=\"managed-api-service.v$PREVIOUS_VERSION\"" config/manifests-$OPERATOR_TYPE/bases/managed-api-service.clusterserviceversion.yaml
  fi
}

create_or_update_csv() {
  "${KUSTOMIZE[@]}" build config/manifests-$OPERATOR_TYPE | operator-sdk generate bundle --kustomize-dir config/manifests-$OPERATOR_TYPE --output-dir bundles/$OLM_TYPE/$VERSION --version $VERSION --default-channel stable --package ${PACKAGE_NAME} --channels stable
}

set_version() {
  if [[ -z "$PREVIOUS_VERSION" ]]; then
    echo "No previous version please set correct values in the Makefile and version/version.go files"
  else
    case $OLM_TYPE in
    "managed-api-service")
      "${SED_INLINE[@]}" -E "s/RHOAM_TAG\s+\?=\s+$PREVIOUS_VERSION/RHOAM_TAG \?= $VERSION/g" Makefile
      "${SED_INLINE[@]}" -E "s/managedAPIVersion\s+=\s+\"$PREVIOUS_VERSION\"/managedAPIVersion = \"$VERSION\"/g" version/version.go
      yq e -i ".channels[0].currentCSV=\"$OLM_TYPE.v$VERSION\"" bundles/$OLM_TYPE/*.package.yaml
      ;;
    "multitenant-managed-api-service")
      yq e -i ".channels[0].currentCSV=\"managed-api-service.v$VERSION\"" bundles/$OLM_TYPE/*.package.yaml
      ;;
    *)
      echo "No version found for install type : $(OLM_TYPE)"
      ;;
    esac
  fi
}

set_descriptions() {
  echo "Updating descriptions"
  yq e -i '.spec.validation.openAPIV3Schema.description="RHOAM is the Schema for the RHOAM API"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmis.yaml
  yq e -i '.spec.validation.openAPIV3Schema.properties.spec.description="RHOAMSpec defines the desired state of Installation"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmis.yaml
  yq e -i '.spec.validation.openAPIV3Schema.properties.status.description="RHOAMStatus defines the observed state of Installation"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmis.yaml
}

# Set the image and containerImage fields in the CSV
# Note that multitenant-managed-api-service still uses the single tenant operator image as they are identical
set_images() {
  case $OLM_TYPE in
  "managed-api-service")
    : "${IMAGE_TAG:=rhoam-v${SEMVER}}"
    yq e -i ".spec.install.spec.deployments.[0].spec.template.spec.containers[0].image=\"quay.io/$ORG/$OLM_TYPE:$IMAGE_TAG\"" bundles/$OLM_TYPE/${VERSION}/manifests/managed-api-service.clusterserviceversion.yaml
    yq e -i ".metadata.annotations.containerImage=\"quay.io/$ORG/$OLM_TYPE:$IMAGE_TAG\"" bundles/$OLM_TYPE/${VERSION}/manifests/managed-api-service.clusterserviceversion.yaml
    ;;
  "multitenant-managed-api-service")
    : "${CONTAINER_IMAGE:=$(yq e ".metadata.annotations.containerImage" bundles/managed-api-service/${VERSION}/manifests/managed-api-service.clusterserviceversion.yaml)}"
    yq e -i ".spec.install.spec.deployments.[0].spec.template.spec.containers[0].image=\"${CONTAINER_IMAGE}\"" bundles/$OLM_TYPE/${VERSION}/manifests/managed-api-service.clusterserviceversion.yaml
    yq e -i ".metadata.annotations.containerImage=\"${CONTAINER_IMAGE}\"" bundles/$OLM_TYPE/${VERSION}/manifests/managed-api-service.clusterserviceversion.yaml
    ;;
  esac
}

set_csv_service_affecting_field() {
  local value=$1
  echo "Update CSV for release $SEMVER to be 'serviceAffecting: $value'"
  yq e -i ".metadata.annotations.serviceAffecting= \"$value\" " bundles/$OLM_TYPE/${VERSION}/manifests/managed-api-service.clusterserviceversion.yaml
}

# Sets the related images in the CSV for RHOAM
set_related_images() {
  # Temporary file to aggregate all images
  tmp_related_images_file="./products/related_images.yaml"
  touch $tmp_related_images_file

  # Add or remove items from exclusion list if you wish for an item to be ignored in the list
  exclusionList=(
    "oc_cli"
    "zync_postgresql"
    "system_mysql"
    "backend_redis"
    "system_redis"
    "postgresql"
  )

  echo "Adding related images to the CSV"

  product_config_path="./products/products.yaml"

  length=$(yq e '.products | length' $product_config_path)

  # Get supported components
  for ((i = 0; i < length; i++)); do
    product_name="$(yq e ".products[$i].name" $product_config_path).$(yq e ".products[$i].version" $product_config_path)"
    product_dir=$(yq e ".products[$i].manifestsDir" $product_config_path)

    if [[ $(yq e ".products[$i].installType" $product_config_path) == *"rhoam"* && $(yq e ".products[$i].quayScan" $product_config_path) == true ]]; then
      # Read component version
      if [[ "$product_dir" == *"observability-operator"* ]]; then
        component_version=$(grep currentCSV manifests/"$product_dir"/*.package.yaml | awk -F "operator." '{print $2}')
      else
        component_version=$(grep currentCSV manifests/"$product_dir"/*.package.yaml | awk -F v '{print $2}')
      fi

      # Path to component csv
      component_csv="./manifests/$product_dir/$component_version/*.clusterserviceversion.yaml"

      # Read component name
      component_name=$(yq e '.metadata.name' $component_csv)

      # Read containers section length
      containerLength=$(yq e '.spec.install.spec.deployments[0].spec.template.spec.containers | length' $component_csv)

      for ((y = 0; y < containerLength; y++)); do
        # Read image from the component version but only select quay.io or redhat.registry
        component_image=$(yq e ".spec.install.spec.deployments[0].spec.template.spec.containers[$y].image" $component_csv | yq e 'select((test("quay.")) or (test("registry.redhat")))')

        # If component image is found, check if the list already contains image pointing to that URL, if not, add it to the list
        if [[ -n "$component_image" ]]; then
          yq e -i ".[\"${product_name}\"] += {\"${component_name}\":\"${component_image}\"}" $tmp_related_images_file
        fi
      done

      # Check if the CSV of the component has the relatedImages set, if it does, populate RHOAM CSV with it.
      relatedImagesLength=$(yq e '.spec.relatedImages | length' $component_csv)

      # Adding generic related images but only if such image does not already exists in the list
      if [[ $relatedImagesLength != 0 ]]; then
        for ((y = 0; y < relatedImagesLength; y++)); do
          excluded=false
          relatedImageName=$(yq e ".spec.relatedImages[$y].name" $component_csv)
          relatedImageURL=$(yq e ".spec.relatedImages[$y].image" $component_csv)

          for excludedItem in ${exclusionList[*]}; do
            # Check if item is on the exclusion list
            if [[ "$excludedItem" == "$relatedImageName" ]]; then
              excluded=true
            fi
          done
          # if item is not on exclusion list and is not already in the images list, add it in.
          if [ "$excluded" != true ]; then
            yq e -i ".[\"${product_name}\"] += {\"${relatedImageName}\":\"${relatedImageURL}\"}" $tmp_related_images_file
          fi
        done
      fi
    fi
  done

  # Additional images
  additional_images_path="./products/additional-images.yaml"
  length=$(yq e '. | length' $additional_images_path)

  # Get supported components
  for ((i = 0; i < length; i++)); do
    # Get the key of the additional image
    productKey=$(yq e ". | keys | .[$i]" $additional_images_path)

    # Get the product version from products.yaml using key
    get_product_version $productKey

    # Use key and version as product name, otherwise just use the key
    if [[ -n $version ]]; then
      product_name="$productKey.$version"
    else
      product_name=$productKey
    fi

    # Loop through each subcomponent on key
    productSubComponentsLength=$(yq e ".$productKey | length" $additional_images_path)

    for ((j = 0; j < productSubComponentsLength; j++)); do
      img_name=$(yq e ".$productKey[$j].name" $additional_images_path)
      img_url=$(yq e ".$productKey[$j].url" $additional_images_path)
      yq e -i ".[\"${product_name}\"] += {\"${img_name}\":\"${img_url}\"}" $tmp_related_images_file
    done
  done

  # Output temporary yaml file as json and set as annotation in csv
  m="$(yq -o=j $tmp_related_images_file)" yq e -i ".metadata.annotations.containerImages= strenv(m)" bundles/$OLM_TYPE/${VERSION}/manifests/managed-api-service.clusterserviceversion.yaml

  # Remove temporary yaml file
  rm $tmp_related_images_file
}

# Using the passed in variable, find the associated version in products.yaml if theres a matching name
get_product_version() {
  productLength=$(yq e '.products | length' $product_config_path)
  for ((k = 0; k < productLength; k++)); do
    product="$(yq e ".products[$k].name" $product_config_path)"
    if [[ "$product" == "$1" ]]; then
      version="$(yq e ".products[$k].version" $product_config_path)"
      break
    fi
  done
}

update_smtp_from() {
  echo "Updating the CSV's 'ALERT_SMTP_FROM' value for multitenant RHOAM"
  yq e -i '(.spec.install.spec.deployments.[0].spec.template.spec.containers[0].env.[] | select(.name == "ALERT_SMTP_FROM").value) |= "noreply-alert@rhmw.io"' bundles/$OLM_TYPE/${VERSION}/manifests/managed-api-service.clusterserviceversion.yaml
}

if [[ -z "$SEMVER" ]]; then
  echo "ERROR: no SEMVER value set"
  exit 1
fi

if [[ $SEMVER =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*)?(\+[0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*)?$ ]]; then
  echo "Valid version string: ${SEMVER}"
else
  echo "Error: Invalid version string: ${SEMVER}"
  exit 1
fi

VERSION=$(echo "$SEMVER" | awk -F - '{print $1}')

# Set sed -i as it's different for mac vs gnu
if [[ $(uname) = Darwin ]]; then
  SED_INLINE=(sed -i '')
else
  SED_INLINE=(sed -i)
fi

update_base_csv
create_or_update_csv

# Update version if needed
if [[ "$VERSION" != "$PREVIOUS_VERSION" ]]; then
  set_version
fi

set_descriptions
set_images
set_related_images

if [[ -n "$SERVICE_AFFECTING" ]]; then
  set_csv_service_affecting_field "$SERVICE_AFFECTING"
fi

if [[ "${PREPARE_FOR_NEXT_RELEASE}" = true ]]; then
  yq e -i ".spec.install.spec.deployments.[0].spec.template.spec.containers[0].image=\"quay.io/$ORG/$OLM_TYPE:master\"" bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
  yq e -i ".metadata.annotations.containerImage=\"quay.io/$ORG/$OLM_TYPE:master\"" bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
fi

# If building bundles for multitenant RHOAM, update the ALERT_SMTP_FROM value for the Developer Sandbox clusters
if [[ "${OLM_TYPE}" == "multitenant-managed-api-service" ]]; then
  update_smtp_from
fi

# Move bundle.Dockerfile to the bundle folder
mv bundle.Dockerfile bundles/$OLM_TYPE/$VERSION

# Ensure code is formatted correctly
"${GOFMT[@]}" -w $(find . -type f -name '*.go' -not -path "./vendor/*")
