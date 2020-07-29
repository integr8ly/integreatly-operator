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

echo -e '<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">' > report.txt
echo -e '<html xmlns="http://www.w3.org/1999/xhtml">' > report.txt
echo -e ' <head>' > report.txt
echo -e '  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />' > report.txt
echo -e '  <title>Nightly pipelines results for week ending at $(date +%Y-%m-%d)\n</title>' > report.txt
echo -e '  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>' > report.txt
echo -e '</head>' > report.txt

for job in "${jobs[@]}";
do
  echo -e "Obtaining nightly pipelines results for $job..."
  echo -e "Pipeline: $job\n" >> report.txt
  
  firstBuildNumber=$(curl -k $JENKINS_URL/$job/api/json --silent | jq -r .firstBuild.number)
  lastBuildNumber=$(curl -k $JENKINS_URL/$job/api/json --silent | jq -r .builds[0].number)

  echo -e "Downloading stage data for $job..."
  curl -k $JENKINS_URL/$job/wfapi/runs --silent > wfapi.json

  for ((buildNumber=$lastBuildNumber; buildNumber >= $firstBuildNumber; buildNumber--))
  do
    rm -f build.json
    curl -k $JENKINS_URL/$job/$buildNumber/api/json --silent > build.json

    errorCheck=$(cat build.json | grep "$ERROR_STRING")

    if [[ $errorCheck == *$ERROR_STRING* ]] ; then
      echo -e "No json data found for build $buildNumber. Skipping..." 
      continue
    fi

    buildTimestamp=$(cat build.json | jq -r .timestamp)/1000

    cat wfapi.json | jq -c ".[] | select( .id|tonumber | contains($buildNumber)) | .stages" > stage.json

    stageData=$(cat stage.json)

    echo $stageData

    if [ $PERIOD_END_TIMESTAMP -gt $buildTimestamp ] ; then
      break
    fi

    causedByDesc=$(cat build.json | jq -r .actions[].causes[0].shortDescription | grep -vwE "null")
    
    if [[ $causedByDesc == *"timer"* ]] ; then
      buildUrl=$(cat build.json | jq -r .url)
      getDateString
      buildResult=$(cat build.json | jq -r .result)
      echo -e "\n\t#$buildNumber\t$buildDate\tResult: $buildResult\tUrl: $buildUrl" >> report.txt

      description=$(cat build.json | jq -r .description)
      if [[ $description != "null" ]] ; then
        echo -e "\t\tNote: $description" >> report.txt
      fi
    fi
  done
done

rm -f build.json
echo "Script completed. See generated file: report.txt"