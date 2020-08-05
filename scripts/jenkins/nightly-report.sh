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

rm -f report.html && touch report.html

echo -e '<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">' >> report.html
echo -e '<html xmlns="http://www.w3.org/1999/xhtml">' >> report.html
echo -e ' <head>' >> report.html
echo -e '  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />' >> report.html
echo -e "  <title>Nightly pipelines results for week ending at $(date +%Y-%m-%d)\n</title>" >> report.html
echo -e '  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>' >> report.html
echo -e ' </head>' >> report.html
echo -e ' <body style="margin: 0; padding: 0;">' >> report.html

for job in "${jobs[@]}";
do
  echo -e "Obtaining nightly pipelines results for $job..."
  echo -e "<h1>Pipeline: $job</h1>" >> report.html
  
  firstBuildNumber=$(curl -k $JENKINS_URL/$job/api/json --silent | jq -r .firstBuild.number)
  lastBuildNumber=$(curl -k $JENKINS_URL/$job/api/json --silent | jq -r .builds[0].number)

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

    if [ $PERIOD_END_TIMESTAMP -gt $buildTimestamp ] ; then
      break
    fi

    causedByDesc=$(cat build.json | jq -r .actions[].causes[0].shortDescription | grep -vwE "null")
    
    if [[ $causedByDesc == *"timer"* ]] ; then
      buildUrl=$(cat build.json | jq -r .url)
      getDateString
      buildResult=$(cat build.json | jq -r .result)

      echo -e "<h2>Build: $buildNumber, $buildDate</h2>" >> report.html
      echo -e "<p>Result: <b>$buildResult</b></p>" >> report.html
      echo -e "<a href="$buildUrl">Link</a>" >> report.html

      description=$(cat build.json | jq -r .description)
      if [[ $description != "null" ]] ; then
        echo -e "<p><b>Note: </b>$description</p>" >> report.html
      fi

      echo -e "<h3>Stage results:</h3>" >> report.html

      echo -e '<table border="1" cellpadding="0" cellspacing="0" width="90%">' >> report.html

      echo -e '<tr>' >> report.html
      echo -e '<td>Stage</td>' >> report.html
      echo -e '<td>Result</td>' >> report.html
      echo -e '<td>Duration</td>' >> report.html
      echo -e '<td>Message</td>' >> report.html
      echo -e '</tr>' >> report.html

      echo -e "Downloading stage data for $job / $buildNumber"
      curl -k $JENKINS_URL/$job/$buildNumber/wfapi/describe --silent > stage.json

      for i in `seq 0 11`;
      do
        echo -e '<tr>' >> report.html
        stageName=$(cat stage.json | jq -r ".stages[$i] | .name")
        echo -e "<td>$stageName</td>" >> report.html

        stageResult=$(cat stage.json | jq -r ".stages[$i] | .status")
        echo -e "<td>$stageResult</td>" >> report.html

        stageDuration=$(cat stage.json | jq -r ".stages[$i] | .durationMillis / 1000")
        echo -e "<td>$stageDuration seconds</td>" >> report.html

        stageError=$(cat stage.json | jq -r ".stages[$i] | .error | .message")
        if [[ $stageError != "null" ]] ; then
          echo -e "<td>$stageError</td>" >> report.html
        else
          echo -e "<td></td>" >> report.html
        fi
        echo -e '</tr>' >> report.html
      done
      echo -e '</table>' >> report.html
      rm -f wfapi.json
    fi
  done
done

echo -e '</body>' >> report.html
echo -e '</html>' >> report.html

rm -f stage.json
rm -f build.json
echo "Script completed. See generated file: report.html"