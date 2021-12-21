#!/usr/bin/env bash

if [[ -z "$OLM_TYPE" ]]; then
  OLM_TYPE="integreatly-operator"
fi

case $OLM_TYPE in
  "integreatly-operator")
    LATEST_VERSION=$(grep $OLM_TYPE packagemanifests/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $2}')
    ;;
  "managed-api-service")
    LATEST_VERSION=$(grep $OLM_TYPE packagemanifests/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $3}')
    ;;
  *)
    echo "Invalid OLM_TYPE set"
    echo "Use \"integreatly-operator\" or \"managed-api-service\""
    exit 1
    ;;
esac

CHANNEL="${CHANNEL:-alpha}"
ORG="${ORG:-integreatly}"
REG="${REG:-quay.io}"
BUILD_TOOL="${BUILD_TOOL:-docker}"
UPGRADE_RHMI="${UPGRADE:-false}"
VERSIONS="${BUNDLE_VERSIONS:-$LATEST_VERSION}"
ROOT=$(pwd)
INDEX_IMAGE=""
CATALOG_SOURCE_INSTALL="${OC_INSTALL:-true}"


start() {
  clean_up
  create_work_area
  copy_bundles
  check_upgrade_install
  generate_bundles
  generate_index
  if [ "$CATALOG_SOURCE_INSTALL" = true ] ; then
  create_catalog_source
  fi
  clean_up
}

create_work_area() {
  printf "Creating Work Area \n"

  cd ./packagemanifests/$OLM_TYPE/
  mkdir temp && cd temp
}

copy_bundles() {
  for i in $(echo $VERSIONS | sed "s/,/ /g")
  do
      printf 'Copying bundle version: \n'$i
      cp -R ../$i ./
  done
}

# Remove the replaces field in the csv to allow for a single bundle install or an upgrade install. i.e.
# The install will not require a previous version to replace.
check_upgrade_install() {
  if [ "$UPGRADE_RHMI" = true ] ; then
    # We can return as the csv will have the replaces field by default
    echo 'Not removing replaces field in CSV'
    return
  fi
  # Get the oldest version, example: VERSIONS="2.5,2.4,2.3" oldest="2.3"
  OLDEST_VERSION=${VERSIONS##*,}

  file=`ls './'$OLDEST_VERSION | grep .clusterserviceversion.yaml`

  sed '/replaces/d' './'$OLDEST_VERSION'/'$file > newfile ; mv newfile './'$OLDEST_VERSION'/'$file
}

# Generates a bundle for each of the version specified or, the latest version if no BUNDLE_VERSIONS  specified
generate_bundles() {
  printf "Generating Bundle \n"

  for VERSION in $(echo $VERSIONS | sed "s/,/ /g")
  do
    cd ./$VERSION
    opm alpha bundle generate -d . --channels $CHANNEL \
        --package integreatly --output-dir bundle \
        --default $CHANNEL

    docker build -f bundle.Dockerfile -t $REG/$ORG/${OLM_TYPE}-bundle:$VERSION .
    docker push $REG/$ORG/${OLM_TYPE}-bundle:$VERSION
    operator-sdk bundle validate $REG/$ORG/${OLM_TYPE}-bundle:$VERSION
    cd ..
  done
}

# calls the push index differently for a single version install or upgrade scenario
generate_index() {
  if [[ " ${VERSIONS[@]} " > 1 ]]; then
    INITIAL_VERSION=${VERSIONS##*,}
    push_index $INITIAL_VERSION
    push_index $VERSIONS
  else
    push_index $VERSIONS
  fi
}

# builds and pushes the index for each version included
push_index() {
  VERSIONS_TO_PUSH=$1
  bundles=""
  for VERSION in $(echo $VERSIONS_TO_PUSH | sed "s/,/ /g")
  do
      bundles=$bundles"$REG/$ORG/${OLM_TYPE}-bundle:$VERSION,"
  done
  # remove last comma
  bundles=${bundles%?}

  NEWEST_VERSION="$( cut -d ',' -f 1 <<< "$VERSIONS_TO_PUSH" )"
  opm index add \
      --bundles $bundles \
      --build-tool ${BUILD_TOOL} \
      --tag $REG/$ORG/${OLM_TYPE}-index:$NEWEST_VERSION

  INDEX_IMAGE=$REG/$ORG/${OLM_TYPE}-index:$NEWEST_VERSION

  printf 'Pushing index image:'$INDEX_IMAGE'\n'

  docker push $INDEX_IMAGE
}

# creates catalog source on the cluster
create_catalog_source() {
  printf 'Creating catalog source '$INDEX_IMAGE'\n'
  cd $ROOT
  oc delete catalogsource rhmi-operators -n openshift-marketplace --ignore-not-found=true
  oc process -p INDEX_IMAGE=$INDEX_IMAGE  -f ./config/olm/catalog-source-template.yml | oc apply -f - -n openshift-marketplace
}

# cleans up the working space
clean_up() {
  printf 'Cleaning up work area \n'
  rm -rf $ROOT/packagemanifests/$OLM_TYPE/temp
}

start