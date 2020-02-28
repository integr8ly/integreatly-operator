#!/usr/bin/env bash
CURRENT_TIMESTAMP=$(date +%s)
NUM_OF_DAYS="${NUM_OF_DAYS:-7}"
PERIOD_END_TIMESTAMP=$(( CURRENT_TIMESTAMP - 86400*NUM_OF_DAYS )) # only interested in builds not older than $NUM_OF_DAYS
ERROR_STRING="Error 404"
declare -a DEFAULT_JOBS=("rhmi-install-addon-flow" "rhmi-install-master")
declare -a jobs
declare -i buildNumber
declare -i firstBuildNumber
declare -i lastBuildNumber
declare -i buildTimestamp
declare buildUrl
declare buildDate
declare buildResult

# get optional comma-separated param of jobs to report and split them into array
IN=$JOBS_TO_REPORT
SPECIFIED_JOBS=(${IN//,/ })
if [ $SPECIFIED_JOBS ] ; 
then
  jobs=("${SPECIFIED_JOBS[@]}")
else
  jobs=("${DEFAULT_JOBS[@]}")
fi

getDateString() {
  case "$OSTYPE" in
    darwin*)   buildDate=$(date -r $buildTimestamp +%Y-%m-%d) ;;
    linux*)  buildDate=$(date -d @$buildTimestamp +%Y-%m-%d) ;;
  esac
}

rm -f report.txt && touch report.txt
echo -e "Nightly pipelines results for week ending at $(date +%Y-%m-%d)\n" > report.txt
for job in "${jobs[@]}";
do
  echo -e "Obtaining nightly pipelines results for $job..."
  echo -e "Pipeline: $job\n" >> report.txt
  
  firstBuildNumber=$(curl $JENKINS_URL/$job/api/json --silent | jq -r .firstBuild.number)
  lastBuildNumber=$(curl $JENKINS_URL/$job/api/json --silent | jq -r .builds[0].number)
  
  for ((buildNumber=$lastBuildNumber; buildNumber > $firstBuildNumber; buildNumber--))
  do
    rm -f build.json
    curl $JENKINS_URL/$job/$buildNumber/api/json --silent > build.json

    errorCheck=$(cat build.json | grep "$ERROR_STRING")

    if [[ $errorCheck == *$ERROR_STRING* ]] ; then
      echo -e "No json data found for build $buildNumber. Skipping..." 
      continue
    fi

    buildTimestamp=$(cat build.json | jq -r .timestamp)/1000
    
    if [ $PERIOD_END_TIMESTAMP -gt $buildTimestamp ] ; then
      break
    fi

    causedByDesc=$(cat build.json | jq -r .actions[].causes[0].shortDescription | grep -vwE "null")
    
    if [[ $causedByDesc == *"timer"* ]] ; then
      buildUrl=$(cat build.json | jq -r .url)
      getDateString
      buildResult=$(cat build.json | jq -r .result)
      echo -e "\t#$buildNumber\t$buildDate\tResult: $buildResult\tUrl: $buildUrl\n" >> report.txt
    fi
  done
done

rm -f build.json
echo "Script completed. See generated file: report.txt"