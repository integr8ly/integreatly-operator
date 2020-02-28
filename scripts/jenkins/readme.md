## Generating report for nightly pipelines
There is one mandatory environment variable required (Jenkins url used to get json data about pipelines):

`JENKINS_URL`

There are also two optional parameters: 

`JOBS_TO_REPORT` - a name or comma-separated names of the jobs to create the report for (without any spaces). If not provided - report will be generated for the two default nightly pipelines 
`NUM_OF_DAYS` - number of days for which the report will be generated. Defaults to 7 days, if not provided

To run the script simply run:

`./nightly-report.sh`

There will be report file generated after successful execution: `report.txt`