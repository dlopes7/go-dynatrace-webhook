#!/usr/bin/env bash

echo "Sending request to localhost:5000"
echo "Response:"
curl localhost:5000/zabbix -d '{"ProblemID": "999", "State": "OPEN", "ProblemDetailsText": "Dynatrace problem notification test run details", "ProblemTitle": "Dynatrace problem notification test run"}'