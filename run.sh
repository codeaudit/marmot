#! /bin/bash
echo "Starting keys"
echo ""
eris services start keys -p
sleep 2

echo "Setting chain name:"
chain_name=toadserver_chn
echo "$chain_name"
echo ""

echo "Making key and genesis file"
eris chains make --chain-type=simplechain $chain_name

echo "Getting address"
echo ""
ADDR=`eris services exec keys "ls /home/eris/.eris/keys/data"`
#ADDR=`eris keys ls --container --quiet` ##TODO quiet flag
echo "$ADDR"
echo ""

echo "Setting pubkey"
echo ""
PUB=`eris keys pub $ADDR`
echo "$PUB"
echo ""

echo "Setting and chain directory:"
chain_dir=$HOME/.eris/chains/$chain_name
echo "$chain_dir"
echo ""

echo "Copying default config to "$chain_dir"/default.toml"
echo ""
cp ~/.eris/chains/default/config.toml $chain_dir/

echo "Starting chain"
echo ""
eris chains new $chain_name --dir $chain_dir -p
sleep 2

PK=${PUB//[^A-Z0-9]/}

echo "Setting TOADSERVER_IPFS_NODES"
#NODES="ip1,ip2,ip3" #give IPs where toadserver is running
NODES="" #give IPs where toadserver is running
echo "$NODES"


echo "Starting toadserver"
echo ""
eris services start toadserver --chain=$chain_name --env "MINTX_CHAINID=$chain_name" --env "MINTX_PUBKEY=$PK"
