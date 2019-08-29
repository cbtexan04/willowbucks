#!/bin/bash
curdir=$(pwd)
rootdir="lambdas"
artifacts="artifacts"

rm -rf $artifacts
mkdir $artifacts

find $rootdir -mindepth 1 -maxdepth 1 -type d -print0 |
	while IFS= read -rd '' dir; do
		pushd $curdir/$dir
		base="$(basename $dir)"
		zip="$base.zip"
		(GOARCH=amd64 GOOS=linux go build -o main && zip $zip main) || exit "failed"
		mv *.zip $curdir/$artifacts
		rm main
		popd $dir
	done
