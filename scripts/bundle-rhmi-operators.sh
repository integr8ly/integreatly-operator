#!/usr/bin/env bash

# Prereq:
# - opm
# - operator-sdk
# - oc (optional if OC_INSTALL is set)
# Function:
# Script creates olm bundle/bundles and index/indexes for RHOAM/RHMI 
# Usage:
# make create/olm/bundle OLM_TYPE=<managed||managed-api> UPGRADE=<true||false> BUNDLE_VERSIONS=<VERSION_n, VERSION_n-X...> ORG=<QUAY ORG> REG=<REGISTRY>
# OLM_TYPE - must be specified, refers to type of operator lifecycle manager type, can either be integreatly-operator (RHMI) or managed-api-service (RHOAM)
# UPGRADE - defaults to false, if upgrade is false the oldest version specified in the BUNDLE_VERSIONS will have it's replaces removed, otherwise, replaces will stay
# BUNDLE_VERSIONS - specifies the versions that are going to have the bundles build. Versions must exists in the bundle/OLM_TYPE folder and must be listed in a descending order
# ORG - organization of where to push the bundles and indexes 
# REG - registry of where to push the bundles and indexes, defaults to quay.io
# BUILD_TOOL - tool used for building the index, defaults to docker
# OC_INSTALL - set to true if you want the catalogue source to be created pointing to the "oldest" version within the versions specified (version must have no replaces field)(must be oc logged in)
# Example:
# make create/olm/bundle OLM_TYPE=managed-api-service UPGRADE=false BUNDLE_VERSIONS=1.16.0,1.15.2 ORG=mstoklus

if [[ -z "$OLM_TYPE" ]]; then
  OLM_TYPE="integreatly-operator"
fi

case $OLM_TYPE in
  "integreatly-operator")
    LATEST_VERSION=$(grep $OLM_TYPE bundles/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $2}')
    ;;
  "managed-api-service")
    LATEST_VERSION=$(grep $OLM_TYPE bundles/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $3}')
    ;;
  *)
    echo "Invalid OLM_TYPE set"
    echo "Use \"integreatly-operator\" or \"managed-api-service\""
    exit 1
    ;;
esac

ORG="${ORG}"
REG="${REG:-quay.io}"
BUILD_TOOL="${BUILD_TOOL:-docker}"
UPGRADE_RHMI="${UPGRADE:-false}"
VERSIONS="${BUNDLE_VERSIONS:-$LATEST_VERSION}"
CATALOG_SOURCE_INSTALL="${OC_INSTALL:-false}"
ROOT=$(pwd)
INDEX=""
OLDEST_IMAGE=""

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
  echo "Index images are: "
  echo $INDEX
}

create_work_area() {
  printf "Creating Work Area \n"
  cd ./bundles/$OLM_TYPE/
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
  file=`ls ./$OLDEST_VERSION/manifests | grep .clusterserviceversion.yaml`

  sed '/replaces/d' './'$OLDEST_VERSION'/manifests/'$file > newfile ; mv newfile './'$OLDEST_VERSION'/manifests/'$file

  OLDEST_IMAGE=$REG/$ORG/${OLM_TYPE}-index:$OLDEST_VERSION
}

# Generates a bundle for each of the version specified or, the latest version if no BUNDLE_VERSIONS  specified
generate_bundles() {
  printf "Generating Bundle \n"

  for VERSION in $(echo $VERSIONS | sed "s/,/ /g")
  do
    pwd
    cd ../../..
    docker build -f ./bundles/$OLM_TYPE/bundle.Dockerfile -t $REG/$ORG/${OLM_TYPE}-bundle:$VERSION --build-arg manifest_path=./bundles/$OLM_TYPE/temp/$VERSION/manifests --build-arg metadata_path=./bundles/$OLM_TYPE/temp/$VERSION/metadata --build-arg version=$VERSION .
    docker push $REG/$ORG/${OLM_TYPE}-bundle:$VERSION
    operator-sdk bundle validate $REG/$ORG/${OLM_TYPE}-bundle:$VERSION
    cd ./bundles/managed-api-service/temp
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

  INDEX="""$INDEX
  $INDEX_IMAGE
  """
}

# creates catalog source on the cluster
create_catalog_source() {
printf 'Creating catalog source '$OLDEST_IMAGE'\n'
  cd $ROOT
  oc delete catalogsource rhmi-operators -n openshift-marketplace --ignore-not-found=true
  oc process -p INDEX_IMAGE=$OLDEST_IMAGE  -f ./config/olm/catalog-source-template.yml | oc apply -f - -n openshift-marketplace
}

# cleans up the working space
clean_up() {
  printf 'Cleaning up work area \n'
  rm -rf $ROOT/bundles/$OLM_TYPE/temp
}

start