#!/usr/bin/env bash
set -e

# MGDAPI-1016 Configure OAuth access token (session) expiration and token
# inactivity timeouts for the OpenShift Dedicated cluster (and also the
# associated SSO identity provider if defined)
#
# Author: Jan Lieskovsky <jlieskov@redhat.com>
#
# Note: Only OpenShift user with 'cluster-admin' role/access to the cluster
# can modify token property definitions. Therefore, this script will work
# correctly only if run under such a user account!

# List of default OAuth clients the timeout settings for which to be updated:
# * console
# * The ones created automatically when starting the OpenShift Dedicated API:
#   https://docs.openshift.com/dedicated/4/authentication/understanding-authentication.html#oauth-token-requests_understanding-authentication
# * Those provisioned by a RHOAM instance
declare -ra defaultOauthClients=(
  "console"
  "openshift-browser-client"
  "openshift-challenging-client"
  "redhat-rhoam-3scale"
  "redhat-rhoam-rhsso"
  "redhat-rhoam-rhssouser"
)

### Definitions of helper function(s)

# Usage
function usage() {
  declare -ra recognizedEnvVars=(
    "DONT_UPDATE_OAUTH_CLIENTS"
    "DONT_UPDATE_OAUTH_SERVER"
    "DONT_UPDATE_SSO_IDP"
    "OAUTH_CLIENTS"
    "SESSION_EXPIRATION_TIMEOUT"
    "TOKEN_INACTIVITY_TIMEOUT"
    "VERBOSE"
  )
  echo
  echo "Set session expiration and token inactivity timeout(s) for OpenShift"
  echo "Dedicated 4 cluster (and also the associated SSO identity provider if"
  echo "configured)."
  echo
  echo "  Number of seconds (without the 's' suffix) is accepted as timeout."
  echo "  Both the comma ',' and space ' ' characters are accepted as"
  echo "  delimiters for the 'OAUTH_CLIENTS' environment variable."
  echo
  echo "Synopsis:"
  # Display synopsis
  # First the supported environment variables, one pair of them per row
  local synopsisRow=''
  for index in $(seq 0 "${#recognizedEnvVars[@]}")
  do
    if [ "$(( index % 2 ))" -eq "0" ]
    then
      if [ "$(( index ))" -ne "0" ]
      then
        printf "\n  %-65s \\" "${synopsisRow}"
        synopsisRow=''
      fi
      printf -v synopsisRow '%-30s]'  "[${recognizedEnvVars[$index]}=.."
    elif [ "$(( index ))" -eq "${#recognizedEnvVars[@]}" ]
    then
      printf "\n  %-65s \\" "${synopsisRow}"
    else
      synopsisRow="${synopsisRow} [${recognizedEnvVars[$index]}=..]"
    fi
  done
  # Followed by the script name and optional options argument
  printf '\n  %s\n' "$0 [OPTION]"
  echo
  # Display options
  echo "Options:"
  echo -e "  -h, --help      Show this help and exit."
  echo -e "  -v, --verbose   Enable verbose output."
  echo
  # Supported environment variables
  echo "The script recognizes the following environment variables:"
  echo
  echo "  * DONT_UPDATE_OAUTH_CLIENTS"
  echo "  Skip updating token timeout settings for OAuth clients defined"
  echo "  within the 'openshift-config' namespace of the OSD cluster."
  echo
  echo "  * DONT_UPDATE_OAUTH_SERVER"
  echo "  Skip updating token timeouts settings for the internal OAuth server"
  echo "  of the OSD cluster."
  echo
  echo "  * DONT_UPDATE_SSO_IDP"
  echo "  Skip updating token timeout settings for the SSO identity provider"
  echo "  associated with the OSD cluster."
  echo
  echo "  * OAUTH_CLIENTS"
  echo "  Enumerated list of OAuth clients, defined in the 'openshift-config'"
  echo "  namespace of the OSD cluster token timeout(s) settings for which"
  echo "  should be updated as part of the script run."
  echo
  echo "  * SESSION_EXPIRATION_TIMEOUT"
  echo "  Defines value to be used to configure session expiration timeout"
  echo "  setting for the OSD cluster and associated SSO identity provider."
  echo
  echo "  * TOKEN_INACTIVITY_TIMEOUT"
  echo "  Defines value to be used to configure token inactivity timeout"
  echo "  setting for the OSD cluster and associated SSO identity provider."
  echo
  echo "  * VERBOSE"
  echo "  Enable verbose output."
  echo
  # Default values
  echo "Default Values:"
  echo
  echo "  Minimum value of token inactivity timeout is 300 seconds (5 minutes)."
  echo "  Defaults to minimum value if not specified."
  echo
  echo "  Session expiration timeout defaults to 3000 seconds (50 minutes)"
  echo "  if not specified (default value of inactivity timeout multiplied"
  echo "  by ten)."
  echo
  echo "  Relevant settings of the following OAuth clients are configured"
  echo "  if not overridden by the 'OAUTH_CLIENTS' environment variable:"
  for client in "${defaultOauthClients[@]}"; do echo "  * '${client}'"; done
  echo
  # Examples
  echo "Examples:"
  echo
  echo "  # Set both timeouts of default OAuth clients to timeout defaults"
  echo "  $0"
  echo
  echo "  # Again update fields of default clients. But this time set session"
  echo "  # expiration to 10 hours and token inactivity to 30 minutes"
  echo "  export SESSION_EXPIRATION_TIMEOUT=36000"
  echo "  export TOKEN_INACTIVITY_TIMEOUT=1800"
  echo "  $0"
  echo
  echo "  # For a specific client set token inactivity to 10 minutes. Keep the"
  echo "  # session expiration to its default value (ten times more than token"
  echo "  # inactivity timeout)"
  echo "  $ unset SESSION_EXPIRATION_TIMEOUT"
  echo "  $ OAUTH_CLIENTS='redhat-rhoam-rhsso' TOKEN_INACTIVITY_TIMEOUT=600 $0"
  echo
  exit 0
}

# Display specified message bordered with box of '#' characters
function show_label() {
  local message="${1?Please specify heading for the label!}"
  local length="$(( ${#message} + 8 ))"
  printf '%0.s#' $(seq 1 $length)
  echo
  echo -e "### $message ###"
  printf '%0.s#' $(seq 1 $length)
  echo
}

# Update token expiration or inactivity property on selected OAuth resource
# (identified by resource type and name) by running 'oc patch' with provided
# path and new value arguments on that resource
function oc_patch_token_property() {
  local resourceType="${1?Please specify OAuth or OAuthClient as a resource!}"
  local resourceName="${2?Please specify the name of the resource!}"
  local resourcePath="${3?Please specify the path to the target location!}"
  local newValue="${4?Please specify the new value of the token property!}"

  # Enumerate supported resource types
  declare -ra allowedResourceTypes=( "oauth" "oauthclient" )

  # OpenShift namespace to operate at
  local namespace="openshift-config"

  # Convert both resource type and name to lowercase
  resourceType="$(echo "${resourceType}" | tr '[:upper:]' '[:lower:]')"
  resourceName="$(echo "${resourceName}" | tr '[:upper:]' '[:lower:]')"

  # Verify provided resource is either OAuth or OAuthClient
  if [[ ! "${allowedResourceTypes[*]}" =~ ${resourceType} ]]
  then
    echo "Unsupported resource type: ${resourceType}!"
    exit 1
  fi

  # Prepare the patch argument for 'oc patch' based on actual arguments
  #
  # NOTE: The following variable is intentionally passing 'newValue' argument
  #       unquoted (without escaped double quotes), so the actual value is
  #       passed as integer type to the underlying oc command. It will be
  #       explicitly converted to string later, where necessary.
  local patch="
        [{
          \"op\":    \"replace\",
          \"path\":  \"${resourcePath}\",
          \"value\": ${newValue}
        }]"

  # Issue the actual 'oc patch' command on the 'openshift-config' namespace
  oc patch "${resourceType}" "${resourceName}" \
    --type=json --namespace="${namespace}" \
    --patch="${patch}"
}

# curl returns exit code of 0 (success) even if HTTP status code wasn't 200 OK
# Thus parse the response and return the HTTP status code and response body
# if the status code was within the 2xx (Successful) range of HTTP status
# codes. Otherwise show an error message and exit with appropriate error
function process_curl_response() {
  local response="${1?Please specify the curl response to process!}"
  # Trim trailing whitespace from status code
  # shellcheck disable=SC2155
  local responseStatusCode=$(sed -n '1s/[[:space:]]\+$//;1p' <<< "${response}")
  # Capture the response body without the empty line delimiter
  # shellcheck disable=SC2155
  local responseBody=$(sed -n '/^.$/,$ {n;p;}' <<< "${response}")
  # A HTTP status code outside of 2xx (Successful) class of status codes
  # was received. Exit with appropriate error message
  if ! grep -q "2[0-9][0-9]" <<< "${responseStatusCode}"
  then
    echo >&2
    echo >&2 "Received a HTTP status code outside of '2xx' range!"
    echo >&2 -e "* Status code:   ${responseStatusCode}"
    echo >&2 -e "* Response body: ${responseBody}"
    return 1
  # curl request succeeded. Return HTTP status & response body
  else
    echo "${responseStatusCode}#${responseBody}"
  fi
}

# Get RH-SSO IDP access token via the 'admin-cli' client
function get_sso_access_token() {
  local keycloakRestUrl="${1?Please specify the SSO REST base path!}"
  local keycloakRealm="${2?Please specify the RH-SSO realm to operate at!}"
  local installationPrefix="${3?Please specify the installation prefix!}"

  echo >&2
  echo >&2 "Requesting access token for '${keycloakUsername}' from SSO IDP.."

  # Operate as SSO administrator
  keycloakUsername="admin"

  # Get OAuth access token for 'master' realm, since using username & password
  # of SSO administrator
  local tokenEndpoint="realms/${keycloakRealm}/protocol/openid-connect/token"

  # Assemble the final URL
  local tokenUrl="${keycloakRestUrl}/${tokenEndpoint}"

  # Determine password for Keycloak administrator
  # shellcheck disable=SC2155
  local keycloakPassword=$(
    oc get secret credential-rhsso -n "${installationPrefix}-rhsso" -o json |
    jq -r .data.ADMIN_PASSWORD |
    base64 --decode
  )
  declare -ra curlOptions=(
    '--data'           'client_id=admin-cli'
    '--data'           'grant_type=password'
    '--data'           "username=${keycloakUsername}"
    '--data-urlencode' "password=${keycloakPassword}"
    '--include'
    '--show-error'
    '--silent'
  )
  if [ "x${VERBOSE}" != x ]
  then
    echo >&2
    echo >&2 "About to issue the following curl request:"
    echo >&2 "curl ${curlOptions[*]} ${tokenUrl}"
  fi
  # shellcheck disable=SC2155
  local curlResponse=$(curl "${curlOptions[@]}" "${tokenUrl}")
  # shellcheck disable=SC2155
  local result=$(process_curl_response "${curlResponse}")
  # shellcheck disable=SC2155
  local httpStatusCode=$(cut -d '#' -f1 <<< "${result}")
  # shellcheck disable=SC2155
  local httpBody=$(cut -d '#' -f2 <<< "${result}")
  # Get OAuth token request returns 'HTTP 200 OK' status to indicate success
  # and (access, refresh) tokens are returned within the HTTP response body
  if grep -q '200 OK' <<< "${httpStatusCode}" && [ "x${httpBody}" != x ]
  then
    # shellcheck disable=SC2155
    local accessToken=$(jq -r .access_token <<< "${httpBody}")
    if [ "x${VERBOSE}" != x ]
    then
      echo >&2
      echo >&2 -e "Got the following response to token request:\n${httpBody}"
      echo >&2
      echo >&2 -e "The OAuth access token itself is:\n${accessToken}"
      echo >&2
      echo >&2 'Done.'
    fi
    echo "${accessToken}"
  fi
}

# Issue a GET query against SSO IDP Admin REST API and return the response --
# JSON representation of a particular object (realm, user, etc.)
function get_sso_object_representation() {
  local keycloakRestUrl="${1?Please specify the SSO REST base path!}"
  local endPoint="${2?Please specify the REST endpoint!}"
  local token="${3?Please specify the access token!}"
  # Optional name of the attribute for which to filter the returned
  # representation of the object if specified
  local name="${4}"

  # Assemble the final URL
  local restUrl="${keycloakRestUrl}/${endPoint}"

  echo >&2
  if [ "x${name}" != x ]
  then
    echo >&2 "Retrieving value of '${name}' field from SSO REST endpoint:"
  else
    echo >&2 "Retrieving JSON from SSO REST endpoint:"
  fi
  echo >&2 "${restUrl}"

  declare -ra curlOptions=(
    '--header' 'Accept: application/json'
    '--header' "Authorization: Bearer ${token}"
    '--include'
    '--show-error'
    '--silent'
  )
  if [ "x${VERBOSE}" != x ]
  then
    echo >&2
    echo >&2 "About to issue the following curl request:"
    echo >&2 "curl ${curlOptions[*]} ${restUrl}"
  fi
  # shellcheck disable=SC2155
  local curlResponse=$(curl "${curlOptions[@]}" "${restUrl}")
  # shellcheck disable=SC2155
  local result=$(process_curl_response "${curlResponse}")
  # shellcheck disable=SC2155
  local httpStatusCode=$(cut -d '#' -f1 <<< "${result}")
  # shellcheck disable=SC2155
  local httpBody=$(cut -d '#' -f2 <<< "${result}")
  # HTTP status code of '200 OK' is in common returned by Keycloak REST API GET
  # request to indicate success. Response body contains the requested object
  if grep -q '200 OK' <<< "${httpStatusCode}" && [ "x${httpBody}" != x ]
  then
    if [ "x${VERBOSE}" != x ]
    then
      echo >&2
      echo >&2 "Got the following response to get object representation:"
      echo >&2 "${httpBody}"
    fi
    # If also the name parameter was specified, filter the returned response to
    # contain only the entry for the particular attribute / field
    if [ "x${name}" != x ]
    then
      result=$(jq ."${name}" <<< "${httpBody}")
      if [ "x${VERBOSE}" != x ]
      then
        echo >&2
        echo >&2 "Object representation filtering with '${name}' was requested."
        echo >&2 "Resulting JSON:"
        echo >&2 "${result}"
      fi
      # Otherwise return complete response
    else
      result=$(jq '.' <<< "${httpBody}")
    fi
    echo "${result}"
  fi
}

# Issue a POST query against SSO IDP Admin REST API to set a particular object
# (realm, user, etc.) to the specified JSON representation
function set_sso_object_representation() {
  local keycloakRestUrl="${1?Please specify the SSO REST base path!}"
  local endPoint="${2?Please specify the REST API endpoint!}"
  local token="${3?Please specify the access token!}"
  # Required parameter - name of the attribute the current value of which is to
  # be replaced with the new value
  local name="${4?Please specify the name of attribute!}"
  # The actual new value to use as a replacement
  local newValue="${5?Please specify the new value of the attribute!}"

  # Assemble the final URL
  local restUrl="${keycloakRestUrl}/${endPoint}"

  echo >&2
  echo >&2 "Issuing an UPDATE request to SSO REST endpoint:"
  echo >&2 "${restUrl}"

  # Get local JSON representation of the specific REST endpoint
  # shellcheck disable=SC2155
  local localJson=$(
    get_sso_object_representation \
    "${keycloakRestUrl}"          \
    "${endPoint}"                 \
    "${token}"
  )
  # Filter current value of the name attribute from it
  # shellcheck disable=SC2155
  local currentValue=$(jq ."${name}" <<< "${localJson}")
  if [ "x${VERBOSE}" != x ]
  then
    echo >&2
    echo >&2 "Current value of '${name}' attribute is: ${currentValue}"
  fi
  # Sanity check: If the current value is numeric, the new one also needs to be
  if [[ "${currentValue}" =~ ^[0-9]+$ ]] && [[ ! "${newValue}" =~ ^[0-9]+ ]]
  then
    echo >&2 "Datatype of current '$name' value is number."
    echo >&2 "Please specify a number!"
    echo >&2 "Received '${newValue}' as the new value argument!"
    exit 1
  fi
  # Modify local JSON representation of the returned SSO object with new value
  # shellcheck disable=SC2155
  local modifiedJson=$(
    # If datatype of current value is number, insert the new value directly
    if [[ "${currentValue}" =~ ^[0-9]+$ ]]
    then
      echo "${localJson}" | jq ".${name}=${newValue}"
    # Otherwise convert it to string first before the actual insert
    else
      echo "${localJson}" | jq ".${name}|=sub(${currentValue};\"${newValue}\")"
    fi
  )
  if [ "x${VERBOSE}" != x ]
  then
    echo >&2
    echo >&2 "Original JSON reply for '${endPoint}' end point:"
    jq   >&2 '.' <<< "${localJson}"
    echo >&2
    echo >&2 "JSON modified with the new '${newValue}' value:"
    jq   >&2 '.' <<< "${modifiedJson}"
  fi
  # Propagate the locally modified JSON representation to the remote SSO IDP
  declare -ra curlOptions=(
    '--data'    "${modifiedJson}"
    '--header'  'Accept: application/json'
    '--header'  "Authorization: Bearer ${token}"
    '--header'  'Content-Type: application/json'
    '--include'
    '--request' 'PUT'
    '--show-error'
    '--silent'
  )
  if [ "x${VERBOSE}" != x ]
  then
    echo >&2
    echo >&2 "About to issue the following curl request:"
    echo >&2 "curl ${curlOptions[*]} ${restUrl}"
  fi
  # shellcheck disable=SC2155
  local curlResponse=$(curl "${curlOptions[@]}" "${restUrl}")
  # shellcheck disable=SC2155
  local result=$(process_curl_response "${curlResponse}")
  # shellcheck disable=SC2155
  local httpStatusCode=$(cut -d '#' -f1 <<< "${result}")
  # shellcheck disable=SC2155
  local httpBody=$(cut -d '#' -f2 <<< "${result}")
  # Per empirical experience HTTP status code of '204 No Content' is returned
  # from Keycloak REST API for request against the 'PUT /{realm}' endpoint to
  # indicate success. Response body is empty in that case
  if grep -q '204 No Content' <<< "${httpStatusCode}"
  then
    if [ "x${VERBOSE}" != x ]
    then
      echo >&2
      echo >&2 "Got the following response to set object representation:"
      echo >&2 "${httpBody}"
    fi
    # Fetch the attribute value from remote SSO IDP again and return it
    # shellcheck disable=SC2155
    local currentRemoteAttributeValue=$(
      get_sso_object_representation \
      "${keycloakRestUrl}"          \
      "${endPoint}"                 \
      "${token}"                    \
      "${name}"
    )
    echo "${currentRemoteAttributeValue}"
  fi
}

# Via Keycloak REST API update specific property of the top-level
# representation of the Keycloak realm
function set_sso_realm_property() {
  local keycloakRestUrl="${1?Please specify the SSO REST base path!}"
  local token="${2?Please specify the OAuth access token!}"
  local keycloakRealm="${3?Please specify the SSO realm!}"
  local property="${4?Please specify the property name!}"
  local value="${5?Please specify the new value of the property!}"

  # Convert specified value to number
  value=$(( value ))

  # Operate as SSO administrator
  keycloakUsername="admin"

  # https://www.keycloak.org/docs-api/12.0/rest-api/index.html#_realms_admin_resource
  #
  # Use 'Realm Admin' resource of Keycloak REST API to get the top-level
  # representation of the realm
  local endPoint="${keycloakUsername}/realms/${keycloakRealm}"

  echo
  echo "* Setting '${property}' of '${keycloakRealm}' to: '${value}' seconds.."
  result=$(
    set_sso_object_representation \
    "${keycloakRestUrl}"          \
    "${endPoint}"                 \
    "${token}"                    \
    "${property}"                 \
    "${value}"
  )

  # Confirm the property got updated correctly
  if [ "x${result}" != x ] && [ "${result}" -eq "${value}" ]
  then
    echo
    echo "Done."
  else
    echo "Failed to update '${property}' value to: '${value}'"
    exit 1
  fi
}

### Main script

# Display usage if requested
if [[ "$#" -eq "1" ]] && [[ "$1" =~ ^-h$ || "$1" =~ ^--help$ ]]
then
  usage
# Enable verbose mode if requested
elif [ "x$*" != x ] && grep -q -- '-v\|--verbose' <<< "$*"
then
  VERBOSE="true"
fi

# Verify executables required by the script are available on the system
declare -ra requiredExecutables=( "base64" "jq" "oc" )

for executable in "${requiredExecutables[@]}"
do
  if ! which "${executable}" >& /dev/null
  then
    echo "Unable to determine the full path of the '${executable}' executable!"
    echo "Please ensure it's installed on the system and the system path is"
    echo "updated accordingly."
    exit 1
  fi
done

# Handle input arguments

# Value of token inactivity timeout in seconds of type integer number without
# the 's' suffix. Optional argument. Minimum allowed value is '300s' per:
#
# https://docs.openshift.com/container-platform/latest/authentication/configuring-internal-oauth.html#oauth-token-inactivity-timeout_configuring-internal-oauth
#
# Defaults to minimum allowed value **without the 's' suffix!** if unset.
# Using integer as a datatype allows to re-use it for both OAuth server and
# clients
TOKEN_INACTIVITY_TIMEOUT="${TOKEN_INACTIVITY_TIMEOUT:-300}"

# Verify provided TOKEN_INACTIVITY_TIMEOUT value is a number greater than 300
if   [[ ! "${TOKEN_INACTIVITY_TIMEOUT}" =~ ^[0-9]{3,}$ ]]
then
  echo "Please specify inactivity timeout in seconds without the 's' suffix!"
  exit 1
elif [[ ! "${TOKEN_INACTIVITY_TIMEOUT}" -ge "300"      ]]
then
  echo "Minimum allowed token inactivity timeout value is 300 (seconds)."
  echo "Please specify a number equal or greater than that!"
  exit 1
fi

# Value of token (session) expiration timeout in seconds of type integer number
# without the 's' suffix. Optional argument. Defaults to value of
# TOKEN_INACTIVITY_TIMEOUT multiplied by 10, if unset.
if [ "x${SESSION_EXPIRATION_TIMEOUT}" == x ]
then
  SESSION_EXPIRATION_TIMEOUT=$(( TOKEN_INACTIVITY_TIMEOUT * 10 ))
fi

# Verify the specified SESSION_EXPIRATION_TIMEOUT value is a number
if   [[ ! "${SESSION_EXPIRATION_TIMEOUT}" =~ ^[0-9]{3,}$ ]]
then
  echo "Please specify expiration timeout in seconds without the 's' suffix!"
  exit 1
fi

# If OpenShift version is v4 and greater set the token expiration timeout for
# internal OAuth server.
#
# Note: Skip this step for OpenShift server of v3, since in that case the
# timeouts need to be configured in the 'oauthConfig' section of the master
# configuration file!
readonly ocServerVersion=$(
  oc version -o yaml |
  grep 'openshiftVersion' |
  # Filter any triplet of numbers separated by a dot '.'
  grep -o '\(\([0-9]\+\.\)\{2\}[0-9]\+\)'
)

echo
echo "Updating OAuth access token settings:"
echo

if [[ "${ocServerVersion}" =~ ^4\..+$ ]]
then
  # Configure timeouts on OAuth server only:
  # * If operating on default OAuthClients
  # * If 'DONT_UPDATE_OAUTH_SERVER' environment variable isn't defined
  if [ "x${OAUTH_CLIENTS}" == x ] && [ "x${DONT_UPDATE_OAUTH_SERVER}" == x ]
  then
    show_label "For OpenShift '$ocServerVersion' cluster. Setting token:"
    echo "* Expiration timeout to: '${SESSION_EXPIRATION_TIMEOUT}' seconds."
    oc_patch_token_property                          \
    "OAuth"                                          \
    "cluster"                                        \
    "/spec/tokenConfig/accessTokenMaxAgeSeconds"     \
    "${SESSION_EXPIRATION_TIMEOUT}"

    echo "* Inactivity timeout to: '${TOKEN_INACTIVITY_TIMEOUT}' seconds."
    # Per .spec.tokenConfig config.openshift.io/v1 OAuth API object the datatype
    # for 'accessTokenInactivityTimeout' property is string, thus append 's' to
    # the specified value and enclose it with double quotes
    oc_patch_token_property                          \
    "OAuth"                                          \
    "cluster"                                        \
    "/spec/tokenConfig/accessTokenInactivityTimeout" \
    "${TOKEN_INACTIVITY_TIMEOUT}s"

    echo
  fi
else
  echo "Failed to determine the version of the OpenShift server!"
  echo "Please login and retry."
  exit
fi

# Since OAuth client settings if configured override the same settings possibly
# present in the internal OAuth server, also configure the token expiration and
# inactivity timeouts for all relevant OAuth clients. Either those specified
# via a dedicated OAUTH_CLIENTS environment variable passed as a script
# argument, or the default ones.

if [ "x${OAUTH_CLIENTS}" != x ]
then
  if [[ "${OAUTH_CLIENTS}" =~ ^.*,.*$ ]]
  then
    OAUTH_CLIENTS=$(echo "${OAUTH_CLIENTS}" | tr ',' ' ')
  fi
  declare -ra clients=( "${OAUTH_CLIENTS}" )
else
  declare -ra clients=( "${defaultOauthClients[@]}" )
fi

# Update token timeouts for aforementioned OAuth clients only if
# 'DONT_UPDATE_OAUTH_CLIENTS' environment variable isn't defined

if [ "x${DONT_UPDATE_OAUTH_CLIENTS}" == x ]
then
  for oauthClient in "${clients[@]}"
  do
    if ! oc get oauthclient/"${oauthClient}" >& /dev/null
    then
      echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
      echo "Skipping configuring timeout settings for '${oauthClient}'"
      echo "OAuth client because it doesn't exist on the OpenShift server"
      echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
    else
      echo
      show_label "For '${oauthClient}' client. Setting token:"

      echo "* Expiration timeout to: '${SESSION_EXPIRATION_TIMEOUT}' seconds."
      oc_patch_token_property                 \
      "OAuthClient"                           \
      "${oauthClient}"                        \
      "/accessTokenMaxAgeSeconds"             \
      "${SESSION_EXPIRATION_TIMEOUT}"

      echo "* Inactivity timeout to: '${TOKEN_INACTIVITY_TIMEOUT}' seconds."
      # Per oauth.openshift.io/v1 OAuthClient API object the datatype of
      # 'accessTokenInactivityTimeoutSeconds' property is integer, no
      # additional processing needed for the specified timeout value
      oc_patch_token_property                 \
      "OAuthClient"                           \
      "${oauthClient}"                        \
      "/accessTokenInactivityTimeoutSeconds"  \
      "${TOKEN_INACTIVITY_TIMEOUT}"

      echo
    fi
  done
fi

# Also propagate both timeout settings to SSO IDP if 'DONT_UPDATE_SSO_IDP'
# environment variable isn't defined
if [ "x${DONT_UPDATE_SSO_IDP}" == x ]
then
  # Use 'https:// as the default URL scheme for SSO web console URL
  urlScheme="https://"
  # Error message to be thrown in the case SSO IDP is not configured yet
  declare -ra ssoNotConfiguredErrorMessage=(
    "Failed to determine the SSO console URL! SSO IDP not configured yet?\n"
    "Run the '$(dirname "$0")/setup-sso-idp.sh' script first, with both the\n"
    "TOKEN_INACTIVITY_TIMEOUT and SESSION_EXPIRATION_TIMEOUT environment\n"
    "variables defined to configure the timeout settings for SSO IDP!"
  )
  # Definition of REALM, INSTALLATION_PREFIX, and KEYCLOAK_URL variables below
  # was taken from `scripts/setup-sso-idp.sh` script
  REALM="${REALM:-testing-idp}"
  INSTALLATION_PREFIX="${INSTALLATION_PREFIX:-$(
    oc get RHMIs --all-namespaces -o json |
    jq -r .items[0].spec.namespacePrefix
  )}"
  INSTALLATION_PREFIX=${INSTALLATION_PREFIX%-} # remove trailing dash
  if ! oc get namespace "${INSTALLATION_PREFIX}-rhsso" >& /dev/null
  then
    # Display the error message without the heading space(s)
    echo -e "${ssoNotConfiguredErrorMessage[@]}" | sed 's/^[ ]//g'
    exit 1
  else
    KEYCLOAK_URL=${urlScheme}$(
      oc get route keycloak-edge -n "${INSTALLATION_PREFIX}-rhsso" -o json |
      jq -r .spec.host
    )
    # Valid SSO console URL contains more than just the URL scheme
    if [ "${KEYCLOAK_URL}" == "${urlScheme}" ]
    then
      # Display the error message without the heading space(s)
      echo -e "${ssoNotConfiguredErrorMessage[@]}" | sed 's/^[ ]//g'
      exit 1
    fi
  fi

  # Operate as SSO administrator
  keycloakUsername="admin"

  # KEYCLOAK_URL input was already verified for sanity above. Check value of
  # REALM and INSTALLATION_PREFIX yet
  if [ "x${REALM}" != x ] && [ "x${INSTALLATION_PREFIX}" != x ]
  then

    # https://www.keycloak.org/docs-api/12.0/rest-api/index.html#_uri_scheme
    #
    # Ensure Keycloak REST base path ends up with '/auth' suffix
    if [[ ! "${KEYCLOAK_URL}" =~ ^.*/auth$ ]]
    then
      keycloakRestBasePath="${KEYCLOAK_URL}/auth"
    fi

    show_label "For SSO IDP at: '${KEYCLOAK_URL}'"

    # Get OAuth access token for 'master' realm since operating under the
    # credentials of SSO administrator
    token=$(
      get_sso_access_token            \
      "${keycloakRestBasePath}"       \
      "master"                        \
      "${INSTALLATION_PREFIX}"
    )
    if [ "x${token}" != x ]
    then
      # https://www.keycloak.org/docs/latest/server_admin/index.html#_timeouts
      #
      # Propagate TOKEN_INACTIVITY_TIMEOUT to 'SSO Session Idle' field of SSO
      property="ssoSessionIdleTimeout"
      set_sso_realm_property          \
      "${keycloakRestBasePath}"       \
      "${token}"                      \
      "${REALM}"                      \
      "${property}"                   \
      "${TOKEN_INACTIVITY_TIMEOUT}"

      # Propagate SESSION_EXPIRATION_TIMEOUT to 'SSO Session Max' field of SSO
      property="ssoSessionMaxLifespan"
      set_sso_realm_property            \
      "${keycloakRestBasePath}"         \
      "${token}"                        \
      "${REALM}"                        \
      "${property}"                     \
      "${SESSION_EXPIRATION_TIMEOUT}"

    else
      echo
      echo "Failed to obtain the OAuth access token for '${keycloakUsername}'!"
      echo "Verify RH-SSO url, API endpoint and user credentials are correct!"
      exit 1
    fi
  fi
fi
