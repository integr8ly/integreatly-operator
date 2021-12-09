#!/usr/bin/env bash
set -e
set -o pipefail

if [[ -z "$OLM_TYPE" ]]; then
  OLM_TYPE="integreatly-operator"
fi

case $OLM_TYPE in
  "integreatly-operator")
    PREVIOUS_VERSION=$(grep $OLM_TYPE bundles/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $2}') || echo "No previous version"
    
    OPERATOR_TYPE=rhmi
    ;;
  "managed-api-service")
    PREVIOUS_VERSION=$(grep $OLM_TYPE bundles/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $3}') || echo "No previous version"
    echo "PREVIOUS VERSION IS ${PREVIOUS_VERSION}"
    OPERATOR_TYPE=rhoam
    ;;
  *)
    echo "Invalid OLM_TYPE set"
    echo "Use \"integreatly-operator\" or \"managed-api-service\""
    exit 1
    ;;
esac

if [[ -z "$ORG" ]]; then
  ORG="integreatly"
else
  ORG="$ORG"
fi

# Optional environment variable to set a different Kustomize path. If this
# variable is not set, it will use the one from the $PATH or install Kustomize
if [[ -z $KUSTOMIZE_PATH ]]; then
  KUSTOMIZE="/usr/local/bin/kustomize"
else
  KUSTOMIZE="/usr/local/bin/kustomize"
fi

# Path to gofmt
if [[ -z $GOROOT ]]; then
  GOFMT="/usr/local/go/bin/gofmt"
else
  GOFMT="$GOROOT/bin/gofmt"
fi

create_new_csv() {

  if [[ -z "$PREVIOUS_VERSION" ]]
    then
      echo "here1..."
      "${KUSTOMIZE[@]}" build config/manifests-$OPERATOR_TYPE | operator-sdk generate bundle --kustomize-dir config/manifests-$OPERATOR_TYPE --output-dir bundles/$OLM_TYPE/$VERSION --version $VERSION --default-channel rhmi 
    else
      echo "here..."
      "${KUSTOMIZE[@]}" build config/manifests-$OPERATOR_TYPE | operator-sdk generate bundle --kustomize-dir config/manifests-$OPERATOR_TYPE --output-dir bundles/$OLM_TYPE/$VERSION --version $VERSION --default-channel rhmi
  fi
}

update_csv() {
  echo "here2..."
  "${KUSTOMIZE[@]}" build config/manifests-$OPERATOR_TYPE | operator-sdk generate bundle --kustomize-dir config/manifests-$OPERATOR_TYPE --output-dir bundles/$OLM_TYPE/$VERSION --version $VERSION --default-channel rhmi 
}

# The base CSV is used to generate the final CSV by combining it with the other operator
# manifests. In operator-sdk v1.2.0, the replaces field of the new CSV is set from
# the current version of **the base CSV**, so we need to update the base CSV in order
# for the replaces field to be set when generating the next release
update_base_csv() {
  yq e -i ".metadata.name=\"$OLM_TYPE.v$VERSION\"" config/manifests-$OPERATOR_TYPE/bases/$OLM_TYPE.clusterserviceversion.yaml
  yq e -i ".spec.version=\"$VERSION\"" config/manifests-$OPERATOR_TYPE/bases/$OLM_TYPE.clusterserviceversion.yaml
  if [[ "${VERSION}" != "${PREVIOUS_VERSION}" ]]; then
    echo "inside if"
    # yq e -i ".spec.replaces=\"${OLM_TYPE}.${PREVIOUS_VERSION}\"" bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
    yq e -i ".spec.replaces=\"$OLM_TYPE.v$PREVIOUS_VERSION\"" config/manifests-$OPERATOR_TYPE/bases/$OLM_TYPE.clusterserviceversion.yaml
  fi
  echo "done with if"
}

set_version() {
  if [[ -z "$PREVIOUS_VERSION" ]]
    then
      echo "No previous version please set correct values in the Makefile and version/version.go files"
    else
      case $OLM_TYPE in
        "integreatly-operator")
          "${SED_INLINE[@]}" -E "s/RHMI_TAG\s+\?=\s+$PREVIOUS_VERSION/RHMI_TAG \?= $VERSION/g" Makefile
          "${SED_INLINE[@]}" -E "s/version\s+=\s+\"$PREVIOUS_VERSION\"/version = \"$VERSION\"/g" version/version.go
          yq e -i ".channels[0].currentCSV=\"$OLM_TYPE.v$VERSION\"" bundles/$OLM_TYPE/*.package.yaml
          ;;
        "managed-api-service")
          "${SED_INLINE[@]}" -E "s/RHOAM_TAG\s+\?=\s+$PREVIOUS_VERSION/RHOAM_TAG \?= $VERSION/g" Makefile
          "${SED_INLINE[@]}" -E "s/managedAPIVersion\s+=\s+\"$PREVIOUS_VERSION\"/managedAPIVersion = \"$VERSION\"/g" version/version.go
          yq e -i ".channels[0].currentCSV=\"$OLM_TYPE.v$VERSION\"" bundles/$OLM_TYPE/*.package.yaml
          ;;
        *)
          echo "No version found for install type : $(OLM_TYPE)"
          ;;
      esac
  fi
}

set_installation_type() {
  if [[ -z "$PREVIOUS_VERSION" ]]
    then
      echo "No previous version please set correct values in the Makefile and version/version.go files"
    else
      case $OLM_TYPE in
        "integreatly-operator")
          echo "using default INSTALLATION_TYPE found in deploy/operator.yaml"
          ;;
        "managed-api-service")
          yq e -i '(.spec.install.spec.deployments[0].spec.template.spec.containers[0].env.[] | select(.name=="INSTALLATION_TYPE") | .value) = "managed-api" ' bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
          ;;
        *)
          echo "No INSTALLATION_TYPE found for install type : $(OLM_TYPE)"
          echo "using default INSTALLATION_TYPE found in deploy/operator.yaml"
          ;;
      esac
  fi
}

set_descriptions() {
  case $OLM_TYPE in
   "integreatly-operator")
      echo "using default descriptions"
      ;;
    "managed-api-service")
      echo "Updating descriptions"
      yq e -i '.spec.validation.openAPIV3Schema.description="RHOAM is the Schema for the RHOAM API"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmis.yaml
      yq e -i '.spec.validation.openAPIV3Schema.properties.spec.description="RHOAMSpec defines the desired state of Installation"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmis.yaml
      yq e -i '.spec.validation.openAPIV3Schema.properties.status.description="RHOAMStatus defines the observed state of Installation"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmis.yaml
      yq e -i '.spec.validation.openAPIV3Schema.description="RHOAMConfig is the Schema for the rhoamconfigs API"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmiconfigs.yaml
      yq e -i '.spec.validation.openAPIV3Schema.properties.spec.description="RHOAMConfigSpec defines the desired state of RHOAMConfig"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmiconfigs.yaml
      yq e -i '.spec.validation.openAPIV3Schema.properties.status.description="RHOAMConfigStatus defines the observed state of RHOAMConfig"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmiconfigs.yaml
      yq e -i '.spec.validation.openAPIV3Schema.properties.status.properties.upgradeAvailable.properties.targetVersion.description="target-version: string, version of incoming RHOAM Operator"' bundles/$OLM_TYPE/${VERSION}/manifests/integreatly.org_rhmiconfigs.yaml

      yq e -i '.spec.customresourcedefinitions.owned[1].description="RHOAM is the Schema for the RHOAM API"' bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
      yq e -i '.spec.customresourcedefinitions.owned[1].displayName="RHOAM installation"' bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml


      yq e -i '.spec.customresourcedefinitions.owned[0].description="RHOAMConfig is the Schema for the rhoamconfigs API"' bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
      yq e -i '.spec.customresourcedefinitions.owned[0].displayName="RHOAMConfig"' bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml

      ;;
  esac
}

set_clusterPermissions() {
  case $OLM_TYPE in
   "integreatly-operator")
      echo "using default permissions"
      ;;
    "managed-api-service")
      echo "Updating permissions"
      yq e -i '.spec.maintainers[0].email="rhoam-support@redhat.com"' bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
      yq e -i '.spec.maintainers[0].name="rhoam"' bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
      ;;
  esac
}

set_images() {
  case $OLM_TYPE in
   "integreatly-operator")
  : "${IMAGE_TAG:=v${SEMVER}}"
  yq e -i ".spec.install.spec.deployments.[0].spec.template.spec.containers[0].image=\"quay.io/$ORG/$OLM_TYPE:$IMAGE_TAG\"" bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
  yq e -i ".metadata.annotations.containerImage=\"quay.io/$ORG/$OLM_TYPE:$IMAGE_TAG\"" bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
  ;;
  "managed-api-service")
   : "${IMAGE_TAG:=rhoam-v${SEMVER}}"
  yq e -i ".spec.install.spec.deployments.[0].spec.template.spec.containers[0].image=\"quay.io/$ORG/$OLM_TYPE:$IMAGE_TAG\"" bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
  yq e -i ".metadata.annotations.containerImage=\"quay.io/$ORG/$OLM_TYPE:$IMAGE_TAG\"" bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
  ;;
  esac
}

set_csv_service_affecting_field() {
  local value=$1
  echo "Update CSV for release $SEMVER to be 'serviceAffecting: $value'"
  yq e -i ".metadata.annotations.serviceAffecting= \"$value\" " bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
}

# Sets the related images in the CSV for RHOAM
set_related_images() {
  # Add or remove items from exclusion list if you wish for an item to be ignored in the list
  exclusionList=(
    "oc_cli"
    "zync_postgresql"
    "system_postgresql"
    "system_mysql"
    "backend_redis"
    "system_redis"
    "postgresql"
  )

  echo "Adding related images to the CSV"
  containerImageField="""[
  """
  length=$(yq e -j ./products/products.yaml| jq -r '.products' | jq length)
  # Get supported components
  for (( i=0; i<${length}; i++))
  do
    product_dir=$(yq e ".products[$i].manifestsDir" ./products/products.yaml)
    if [[ $(yq e ".products[$i].installType" ./products/products.yaml) == *"rhoam"* && $(yq e ".products[$i].quayScan" ./products/products.yaml ) == true ]]; then
      # Read component version
      if [[ "$product_dir" == *"observability-operator"* ]]; then
      component_version=$(grep currentCSV manifests/$product_dir/*.package.yaml | awk -F  "operator." '{print $2}')
      else
      component_version=$(grep currentCSV manifests/$product_dir/*.package.yaml | awk -F v '{print $2}')
      fi
      # Read component name
      component_name=$(yq e -j ./manifests/$product_dir/${component_version}/*.clusterserviceversion.yaml | jq '.metadata.name' | tr -d '"')

      # Read containers section length
      containerLength=$(yq e -j ./manifests/$product_dir/${component_version}/*.clusterserviceversion.yaml | jq '.spec.install.spec.deployments[0].spec.template.spec.containers' | jq length)
      for (( y=0; y<$containerLength; y++))
      do
        # Read image from the component version but only select quay.io or redhat.registry
        component_image=$(yq e -j ./manifests/$product_dir/${component_version}/*.clusterserviceversion.yaml | jq ".spec.install.spec.deployments[0].spec.template.spec.containers[$y].image" | jq -r 'select((test("quay.")) or (test("registry.redhat")))' | tr -d '"')

        # If component image is found, check if the list already contains image pointing to that URL, if not, add it to the list
        if [[ ! -z "$component_image" ]]; then
          if [[ "$containerImageField" != *"$component_image"* ]]; then
            containerImageField="$containerImageField{\"component_name\":\"${component_name}\",\"component_url\":\"${component_image}\"},"
          fi
        fi
      done

      # Check if the CSV of the component has the relatedImages set, if it does, populate RHOAM CSV with it.
      relatedImagesLength=$(yq e -j ./manifests/$product_dir/${component_version}/*.clusterserviceversion.yaml | jq -r '.spec.relatedImages' | jq length)
      
      # Adding generic related images but only if such image does not already exists in the list
      if [[ $relatedImagesLength != 0 ]]; then
        for (( y=0; y<$relatedImagesLength; y++))
        do
          excluded=false
          relatedImageName=$(yq e -j ./manifests/$product_dir/${component_version}/*.clusterserviceversion.yaml | jq -r ".spec.relatedImages[$y].name")
          relatedImageURL=$(yq e -j ./manifests/$product_dir/${component_version}/*.clusterserviceversion.yaml | jq -r ".spec.relatedImages[$y].image")
    
          for excludedItem in ${exclusionList[*]}
            do
              # Check if item is on the exclusion list
              if [[ "$excludedItem" == "$relatedImageName" ]]; then
                excluded=true
              fi
          done
          # if item is not on exclusion list and is not already in the images list, add it in.
          if [ "$excluded" != true ]; then
            if [[ "$containerImageField" != *"$relatedImageURL"* ]]; then
              containerImageField="$containerImageField{\"component_name\":\"${relatedImageName}\",\"component_url\":\"${relatedImageURL}\"},"
            fi
          fi
        done
      fi
    fi
  done

  containerImageRemovedLastCharacter=$(echo "${containerImageField::-1}")
  containerImageField="$containerImageRemovedLastCharacter]"
  printf -v m "$containerImageField" ; m="$m" yq e -i ".metadata.annotations.containerImages= strenv(m)" bundles/$OLM_TYPE/${VERSION}/manifests/$OLM_TYPE.clusterserviceversion.yaml
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

# The `projectName` field in the PROJECT file is used by the operator-sdk CLI
# to generate the CSV. In order to be compatible with both types of CSVs
# (RHMI and RHOAM), we need to temporarily set the `projectName` to the desired
# OLM type, and save the current value in order to reset it when we're done
current_project_name=$(yq e '.projectName' PROJECT)
yq e -i ".projectName=\"$OLM_TYPE\"" PROJECT

update_base_csv

# We have a new version so generate the csv
if [[ "$VERSION" != "$PREVIOUS_VERSION" ]]; then
  create_new_csv
  set_version
else
  update_csv
fi

set_installation_type
set_descriptions
set_clusterPermissions
set_images

if [[ -n "$SERVICE_AFFECTING" ]]; then
 set_csv_service_affecting_field "$SERVICE_AFFECTING"
fi

# The following is disabled to unblock rc1 cut of 1.13.0 - it should be renabled before final release.
if [[ "${OLM_TYPE}" == "managed-api-service" ]]; then
 set_related_images
fi

# Reset the project name
yq e -i ".projectName=\"$current_project_name\"" PROJECT

# Ensure the RHMI package is `integreatly`: The operator-sdk CLI will take the
# package name from the PROJECT file, so in the case of RHMI it will set it
# incorrectly to `integreatly-operator`
yq e -i '.packageName="integreatly"' bundles/integreatly-operator/integreatly-operator.package.yaml

# Ensure code is formatted correctly
"${GOFMT[@]}" -w `find . -type f -name '*.go' -not -path "./vendor/*"`
