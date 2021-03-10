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
H1='Accept: application/json'

echo 'Downloading data, this may take awhile...'
curl -s -H "$H1" ${HYPERFOIL_URL}/run/${RUN}/stats/all | jq . > Backup.json


cat Backup.json | jq '.stats[].total.summary.invalid' > failed.txt
cat Backup.json | jq '.stats[].total.summary.requestCount' > total.txt


failed=$(awk '{s+=$1} END {print s}' failed.txt)


total=$(awk '{s+=$1} END {print s}' total.txt)

one_percent=$(expr $total / $hundred)

awk "BEGIN {printf \"Percentage passed = %.4f\n\", ${hundred}-(${failed}/${one_percent}) }"
echo " "
awk "BEGIN {printf \"Percentage failed = %.6f\n\", ${failed}/${one_percent} }"
echo 'Total = ' $total
echo 'Failed = ' $failed
echo 'One percent = '$one_percent
echo " "
cat Backup.json | jq '.stats[] | select(.phase | test("steady")) | select(.metric| test("post.")) | .total.summary.percentileResponseTime."90.0"' > post-percentile
cat Backup.json | jq '.stats[] | select(.phase | test("steady")) | select(.metric| test("get.")) | .total.summary.percentileResponseTime."90.0"' > get-percentile
cat Backup.json | jq '.stats[] | select(.phase | test("steady")) | select(.metric| test("create.")) | .total.summary.percentileResponseTime."90.0"' > login-percentile
for percentile in post-percentile get-percentile login-percentile
do
    NoOfEndpoints=$(wc -l ${percentile} | awk '{print $1}')
    SumPercential=$(awk '{s+=$1} END {print s}' ${percentile})
    mean=$(expr $SumPercential / $NoOfEndpoints)
    max=$(grep -Eo '[0-9]+' ${percentile} | sort -rn | head -n 1)
    echo ${percentile}" Mean = "$mean"ns Max = "$max"ns"
done

# clean up file created as part of the run
rm Backup.json failed.txt get-percentile login-percentile post-percentile total.txt
