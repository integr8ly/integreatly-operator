#!/bin/bash
# About: gets the percentage successful/failed requests and the mean and max 90% percentile time in ms from hyperfoil runs
# Usage:
#
# "export HYPERFOIL_URL=http://hyperfoil-hyperfoil.apps.mwcollabaz-byoc.2mv4.s1.devshift.org"
#./hyperfoil-request-stats.sh <RUN>
#
# run number e.g. 0001
# ./hyperfoil-request-stats.sh 0001
#

RUN=$1
hundred=100

curl -s ${HYPERFOIL_URL}/run/${RUN}/stats/total | jq '.statistics[].summary.invalid' > failed.txt
curl -s ${HYPERFOIL_URL}/run/${RUN}/stats/total | jq '.statistics[].summary.requestCount' > total.txt


failed=$(awk '{s+=$1} END {print s}' failed.txt)
echo 'failed = ' $failed
total=$(awk '{s+=$1} END {print s}' total.txt)
echo 'total = ' $total


one_percent=$(expr $total / $hundred)
echo 'one percent = '$one_percent

percent_failed=$(expr $failed / $one_percent)
percent_passed=$(expr $hundred - $percent_failed)
echo "==============================================================================================="
echo " "
echo 'Percentage failed = ' $percent_failed"%"
echo 'Percentage passed = ' $percent_passed"%"
echo " "
echo '==============================================================================================='
echo '90% Percentile'
echo " "
curl -s ${HYPERFOIL_URL}/run/${RUN}/stats/total| jq '.statistics[] | select(.phase | test("steady.")) | select(.metric | test("post.")) | .summary.percentileResponseTime."90.0"' > post-percentile
curl -s ${HYPERFOIL_URL}/run/${RUN}/stats/total| jq '.statistics[] | select(.phase | test("steady.")) | select(.metric | test("get.")) | .summary.percentileResponseTime."90.0"' > get-percentile
curl -s ${HYPERFOIL_URL}/run/${RUN}/stats/total| jq '.statistics[] | select(.phase | test("steady.")) | select(.metric | test("create.")) | .summary.percentileResponseTime."90.0"' > login-percentile
for percentile in post-percentile get-percentile login-percentile
do
    NoOfEndpoints=$(wc -l ${percentile} | awk '{print $1}')
    SumPercential=$(awk '{s+=$1} END {print s}' ${percentile})
    mean=$(expr $SumPercential / $NoOfEndpoints)
    max=$(grep -Eo '[0-9]+' ${percentile} | sort -rn | head -n 1)
    echo ${percentile}" Mean = "$mean"ns Max = "$max"ns"
done
curl -s ${HYPERFOIL_URL}/run/${RUN}/stats/total | jq '.' > Backup.json