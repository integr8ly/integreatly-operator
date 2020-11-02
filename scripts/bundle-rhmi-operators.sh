#!/usr/bin/env bash

if [[ -z "$OLM_TYPE" ]]; then
  OLM_TYPE="integreatly-operator"
fi

case $OLM_TYPE in
  "integreatly-operator")
    LATEST_VERSION=$(grep $OLM_TYPE deploy/olm-catalog/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $2}')
    PACKAGE="integreatly"
    ;;
  "managed-api-service")
    LATEST_VERSION=$(grep $OLM_TYPE deploy/olm-catalog/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $3}')
    PACKAGE="managed-api-service"
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


start() {
  # TODO: validate input
  create_work_area
  copy_bundles
  check_upgrade_install
  generate_bundles
  generate_index
  create_catalog_source
  clean_up
}

create_work_area() {
  printf "Creating Work Area \n"

  cd ./deploy/olm-catalog/$OLM_TYPE/
  mkdir temp && cd temp
}

copy_bundles() {
  for i in $(echo $VERSIONS | sed "s/,/ /g")
  do
      printf 'Copying bundle version: \n'$i
      cp -R ../$i ./
  done
}

# Remove the replaces field in the csv to allow for a single bundle install. i.e.
# The install will not require a previous version to replace.
check_upgrade_install() {
  if [ "$UPGRADE_RHMI" = true ] ; then
    # We can return as the csv will have the replaces field by default
    echo 'Not replacing rhmi operator'
    return
  fi

  echo "versions:::: "$VERSIONS
  # Get the oldest version, example: VERSIONS="2.5,2.4,2.3" oldest="2.3"
  OLDEST_VERSION=${VERSIONS##*,}

  printf "Removing replaces field from CSV \n"
  file='./'${OLDEST_VERSION}'/'${OLM_TYPE}'.v'${OLDEST_VERSION}'.clusterserviceversion.yaml'
  sed '/replaces/d' $file > newfile ; mv newfile $file
}

generate_bundles() {
  printf "Generating Bundle \n"

  for VERSION in $(echo $VERSIONS | sed "s/,/ /g")
  do
    cd ./$VERSION
    opm alpha bundle generate -d . --channels $CHANNEL \
        --package integreatly --output-dir bundle \
        --default $CHANNEL

    cd ./bundle/

    opm alpha bundle build --directory ./manifests --tag $REG/$ORG/${PACKAGE}-bundle:${VERSION} \
        --package integreatly --channels $CHANNEL --default $CHANNEL

    docker push $REG/$ORG/${PACKAGE}-bundle:$VERSION
    operator-sdk bundle validate $REG/$ORG/${PACKAGE}-bundle:$VERSION

    cd ../../
  done
}

generate_index() {

  bundles=""
  for VERSION in $(echo $VERSIONS | sed "s/,/ /g")
  do
      bundles=$bundles"$REG/$ORG/${PACKAGE}-bundle:$VERSION,"
  done
  # remove last comma
  bundles=${bundles%?}

  NEWEST_VERSION="$( cut -d ',' -f 1 <<< "$VERSIONS" )"
  opm index add \
      --bundles $bundles \
      --build-tool ${BUILD_TOOL} \
      --tag $REG/$ORG/${PACKAGE}-index:$NEWEST_VERSION

  INDEX_IMAGE=$REG/$ORG/${PACKAGE}-index:$NEWEST_VERSION

  printf 'Pushing index image:'$INDEX_IMAGE'\n'

  docker push $INDEX_IMAGE
}

create_catalog_source() {
  printf 'Creating catalog source '$INDEX_IMAGE'\n'
  cd $ROOT
  oc delete catalogsource rhmi-operators -n openshift-marketplace --ignore-not-found=true
  oc process -p INDEX_IMAGE=$INDEX_IMAGE  -f ./deploy/catalog-source-template.yml | oc apply -f - -n openshift-marketplace
}

clean_up() {
  printf 'Cleaning up work area \n'
  rm -rf $ROOT/deploy/olm-catalog/$OLM_TYPE/temp
}

start