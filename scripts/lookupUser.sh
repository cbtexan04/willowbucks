#!/bin/bash
source scripts/.env
curl "https://slack.com/api/users.info?user=$1&token=$ACCESS_TOKEN" | jq .
