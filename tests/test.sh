#!/usr/bin/env bash
# ----------------------------------------------------------
# PURPOSE

# This is the test manager for marmot. It will run the testing
# sequence for marmot using docker.

# ----------------------------------------------------------
# REQUIREMENTS

# eris installed locally

# ----------------------------------------------------------
# USAGE

# test.sh

# ----------------------------------------------------------
# Set defaults

# Where are the Things?

name=marmot
base=github.com/eris-ltd/$name
repo=`pwd`
if [ "$CIRCLE_BRANCH" ]
then
  ci=true
  linux=true
elif [ "$TRAVIS_BRANCH" ]
then
  ci=true
  osx=true
elif [ "$APPVEYOR_REPO_BRANCH" ]
then
  ci=true
  win=true
else
  repo=$GOPATH/src/$base
  ci=false
fi

branch=${CIRCLE_BRANCH:=master}
branch=${branch/-/_}
branch=${branch/\//_}

# Other variables
if [[ "$(uname -s)" == "Linux" ]]
then
  uuid=$(cat /proc/sys/kernel/random/uuid | tr -dc 'a-zA-Z0-9' | fold -w 12 | head -n 1)
elif [[ "$(uname -s)" == "Darwin" ]]
then
  uuid=$(uuidgen | tr -dc 'a-zA-Z0-9' | fold -w 12 | head -n 1  | tr '[:upper:]' '[:lower:]')
else
  uuid="62d1486f0fe5"
fi

was_running=0
test_exit=0
chains_dir=$HOME/.eris/chains
chain_name=marmot-tests-$uuid
name_full="$chain_name"_full_000
chain_dir=$chains_dir/$chain_name

export ERIS_PULL_APPROVE="true"
export ERIS_MIGRATE_APPROVE="true"

# ---------------------------------------------------------------------------
# Needed functionality

ensure_running(){
  if [[ "$(eris services ls -qr | grep $1)" == "$1" ]]
  then
    echo "$1 already started. Not starting."
    was_running=1
  else
    echo "Starting service: $1"
    eris services start $1 1>/dev/null
    early_exit
    sleep 3 # boot time
  fi
}

early_exit(){
  if [ $? -eq 0 ]
  then
    return 0
  fi

  echo "There was an error duing setup; keys were not properly imported. Exiting."
  if [ "$was_running" -eq 0 ]
  then
    if [ "$ci" = true ]
    then
      eris services stop keys
    else
      eris services stop -rx keys
    fi
  fi
  exit 1
}

test_setup(){
  echo "Getting Setup"
  if [ "$ci" = true ]
  then
    eris init --yes --pull-images=true --testing=true 1>/dev/null
    curl -X GET https://raw.githubusercontent.com/eris-ltd/eris-services/master/marmot.toml -o $HOME/.eris/services/marmot.toml # servDef not added yes to init sequence. could import...
  fi
  ensure_running keys

  # make a chain
  eris chains make --account-types=Full:1 $chain_name 1>/dev/null
  key1_addr=$(cat $chain_dir/accounts.csv | grep $name_full | cut -d ',' -f 1)
  echo -e "Default Key =>\t\t\t\t$key1_addr"
  eris chains new $chain_name --dir $chain_dir 1>/dev/null
  sleep 5 # boot time
  echo "Setup complete"
}

perform_tests(){
  # need pubkey for toadserver
  addr=$(eris services exec keys "ls /home/eris/.eris/keys/data")
  PUBKEY=$(eris keys pub $addr)
  echo 
  eris services start toadserver --chain=$chain_name --env "MINTX_PUBKEY=$PUBKEY" --env "MINTX_CHAINID=$chain_name"
  tsIP=$(eris services inspect toadserver NetworkSettings.IPAddress)
  ## XXX need to whitelist the IP making requests on google
  eris services start marmot --env "CLOUD_VISION_API_KEY=$CLOUD_VISION_API_KEY" --env "TOADSERVER_HOST=$tsIP"
  if [ $? -ne 0 ]
  then
    test_exit=1
    return 1
  fi
  echo
  
## need dm ip
  echo "Getting docker-machine IP:"
  dm_active=$(docker-machine active)
  dm_ip=$(docker-machine ip $dm_active)
  echo "$dm_ip"
 
  echo "Constructing URL:"
  url="http://${dm_ip}:2332/postImage/dougdocker.png"
  echo "$url"
  
  echo "Posting dougdocker.png to marmot checker"
  curl --silent -X POST $url --data-binary "@dougdocker.png"
  if [ $? -ne 0 ]
  then
    test_exit=1
    return 1
  fi
  sleep 10 # let all the things happen

  # ask toadserver for the file
  # XXX need to get that unique image id added (or fix with temp file, maybe?

  echo "Getting dougdocker.png from toadserver"
  curl --silent -X GET http://${dm_ip}:11113/getfile/dougdocker.png -o dougtest.png
  if [ $? -ne 0 ]
  then
    test_exit=1
    return 1
  fi
  echo

  # compare $image_file with img in pwd.
  echo "Converting both images to base64"
  got=$(base64 dougtest.png)
  have=$(base64 dougdocker.png)

  echo "Comparing images"
  if [ $got != $have ]; then
    echo "The post image does not match doug"
    test_exit=1
    return 1
  fi
}

test_teardown(){
  if [ "$ci" = false ]
  then
    echo
    eris services stop -rxf marmot 1>/dev/null
    eris services stop -rxf toadserver 1>/dev/null
    eris chains stop -f $chain_name 1>/dev/null
    eris chains rm -x --file $chain_name 1>/dev/null
    if [ "$was_running" -eq 0 ]
    then
      eris services stop -rx keys &>/dev/null
    fi
    rm dougtest.png
    rm -rf $HOME/.eris/scratch/data/marmot-tests-*
    rm -rf $chain_dir
  else
    eris services stop -f marmot 1>/dev/null
    eris chains stop -f $chain_name 1>/dev/null
  fi
  echo
  if [ "$test_exit" -eq 0 ]
  then
    echo "Tests complete! Tests are Green. :)"
  else
    echo "Tests complete. Tests are Red. :("
  fi
  cd $start
  exit $test_exit
}

# ---------------------------------------------------------------------------
# Get the things build and dependencies turned on

echo "Hello! I'm the marmot that tests marmot."
start=`pwd`
cd $repo
echo ""
echo "Building marmot in a docker container."
set -e
tests/build_tool.sh 1>/dev/null
set +e
if [ $? -ne 0 ]
then
  echo "Could not build marmot. Debug via by directly running [`pwd`/tests/build_tool.sh]"
  exit 1
fi
echo "Build complete."
echo ""

# ---------------------------------------------------------------------------
# Setup

test_setup

# ---------------------------------------------------------------------------
# Go!

perform_tests

# ---------------------------------------------------------------------------
# Cleaning up

test_teardown

