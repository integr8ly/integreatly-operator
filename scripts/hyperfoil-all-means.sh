#!/bin/bash
# About: gets different mean times of the get and post requests in the "steady" phase.
#  Output results as a table with each row representing one run.
# Usage:
#
# "export HYPERFOIL_URL=http://hyperfoil-hyperfoil.apps.mwcollabaz-byoc.2mv4.s1.devshift.org"
#./hyperfoil-request-stats.sh <RUN_START> <RUN_END>
#
# run numbers must be in hexadecimal:
# ./hyperfoil-request-stats.sh 0x0000 0x000A
#

HYPERFOIL_URL="${HYPERFOIL_URL:-http://localhost:8090}"
RUN_START=$1
RUN_END=$2
HEADER='Accept: application/json'

types=("get" "post")
means=("50.0" "90.0" "99.0" "99.9" "99.99")

echo -ne "RUN #\t"
for type in "${types[@]}"
do
  echo -ne "${type}\t"
  for percentile in "${means[@]}"
  do
    echo -ne "${percentile}\t"
  done
done
echo -e "Benchmark\tDescription"

for ((i=RUN_START;i<=RUN_END;i++)); do
  RUN=$(printf "%04X\n" $i)
  curl -s -H "$HEADER" ${HYPERFOIL_URL}/run/${RUN}/stats/all | jq -c '.stats[]' > /tmp/hf-tmp-stats-data.json

  echo -ne "$RUN\t"
  for type in "${types[@]}"
  do
    echo -ne "\t"
    for percentile in "${means[@]}"
    do
      mean=$(cat /tmp/hf-tmp-stats-data.json | jq -r "select(.phase | test(\"steady\")) | select(.metric| test(\""${type}"\")) | .total.summary | \"\(.percentileResponseTime.\"${percentile}\" / 1000000) \(.responseCount)\"" | awk '{w = w + $2; e = e + $1 * $2;} END {printf "%.3f", e/w}')
      echo -ne "${mean}\t"
    done
  done

  curl -s -H "$HEADER" ${HYPERFOIL_URL}/run/${RUN} | jq -r '"\(.benchmark)\t\(.description)"'
done
echo ""