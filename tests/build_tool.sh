#!/usr/bin/env bash
# ----------------------------------------------------------
# PURPOSE

# This is the build script for marmot. It will build the tool
# into docker containers in a reliable and predicatable
# manner.

# ----------------------------------------------------------
# REQUIREMENTS

# docker installed locally

# ----------------------------------------------------------
# USAGE

# build_tool.sh

# ----------------------------------------------------------
# Set defaults

if [ "$CIRCLE_BRANCH" ]
then
  repo=`pwd`
else
  repo=$GOPATH/src/github.com/eris-ltd/marmot
fi

testimage=${testimage:="quay.io/eris/marmot"}
otherimage=${otherimage:="quay.io/eris/toadserver"}

cd $repo

# ---------------------------------------------------------------------------
# Go!
echo "Building $testimage"
docker build -t $testimage:latest .
cd $GOPATH/src/github.com/eris-ltd/toadserver
echo "Building $otherimage"
docker build -t $otherimage:latest .
cd $repo
