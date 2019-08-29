#!/bin/bash
GOARCH=amd64 GOOS=linux go build -o main && zip main.zip main
