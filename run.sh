#! /bin/bash
echo "Starting keys"
echo ""
eris services start keys -p
sleep 2


echo "Setting chain name:"
CHAIN_NAME=toadserver_chn
echo "$CHAIN_NAME"
echo ""

echo "Setting service name:"
SERVICE_NAME=toadserver_srv
echo "$SERVICE_NAME"
echo ""

echo "Making key and genesis file"
eris chains make --chain-type=simplechain $CHAIN_NAME

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
CHAIN_DIR=$HOME/.eris/chains/$CHAIN_NAME
echo "$CHAIN_DIR"
echo ""

echo "Copying default config to "$CHAIN_DIR"/default.toml"
echo ""
cp ~/.eris/chains/default/config.toml $CHAIN_DIR/

echo "Starting chain"
echo ""
eris chains new $CHAIN_NAME --dir $CHAIN_DIR -p
sleep 2

echo "Setting service definition file in:"
echo "$HOME/.eris/services/${SERVICE_NAME}.toml"
echo ""

PK=${PUB//[^A-Z0-9]/}

echo "Setting TOADSERVER_IPFS_NODES"
#NODES="ip1,ip2,ip3" #give IPs where toadserver is running
NODES="" #give IPs where toadserver is running
echo "$NODES"

read -r -d '' SERV_DEF << EOM
name = "$SERVICE_NAME"
chain = "\$chain:toad:l"

[service]
name = "$SERVICE_NAME"
image = "quay.io/eris/toadserver:latest"
ports = [ "11113:11113" ]
volumes = [  ]
environment = [  
"MINTX_NODE_ADDR=http://toad:46657/",
"MINTX_CHAINID=$CHAIN_NAME", 
"MINTX_SIGN_ADDR=http://keys:4767",
"MINTX_PUBKEY=$PK",
"ERIS_IPFS_HOST=http://ipfs",
"TOADSERVER_IPFS_NODES=$NODES"
]

[dependencies]
services = [ "ipfs", "keys" ]

[maintainer]
name = "Eris Industries"
email = "support@erisindustries.com"

[location]
repository = "github.com/eris-ltd/toadserver"

[machine]
include = [ "docker" ]
requires = [ "" ]
EOM

echo "$SERV_DEF" > "$HOME/.eris/services/${SERVICE_NAME}.toml"

echo "Starting toadserver"
echo ""
eris services start $SERVICE_NAME --chain=$CHAIN_NAME
sleep 5




<< COMMENT
echo "------------ Beginning test sequence -----------:"
echo "Generating test file"
echo ""
FILE_CONTENTS_POST="testing the toadserver"
FILE_NAME=hungryToad.txt
FILE_PATH=$CHAIN_DIR/$FILE_NAME

echo "$FILE_CONTENTS_POST" > $FILE_PATH

echo "--------POSTING to toadserver------------"
echo ""
#TODO use services ports?

## wake up ipfs
F_CONTENTS_POST="work pls ipfs"
F_NAME=guh.txt
F_PATH=$CHAIN_DIR/$FILE_NAME
echo "$F_CONTENTS_POST" > $F_PATH
eris files put $F_PATH

curl --silent -X POST http://0.0.0.0:11113/postfile/${FILE_NAME} --data-binary "@$FILE_PATH"

echo "Sleeping for 5 seconds: let IPFS boot & nameReg transaction confirm."
echo "."
sleep 1
echo ".."
sleep 1
echo "..."
sleep 1
echo "...."
sleep 1
echo "....."
sleep 1
echo ""

echo "----------GETING from toadserver-----------"
echo ""

FILE_CONTENTS_GET=$(curl --silent -X GET http://0.0.0.0:11113/getfile/${FILE_NAME}) #output directly or use -o to save to file & read

echo "Comparing posted content with getted content"
echo ""
if [[ "$FILE_CONTENTS_POST" != "$FILE_CONTENTS_GET" ]]; then
	echo "FAIL"
	echo "GOT $FILE_CONTENTS_GET"
	echo "EXPECTED $FILE_CONTENTS_POST"
else
	echo "PASS"
fi

echo ""
echo "-------------TEARDOWN-----------------"
echo ""
echo "Kill & Remove Services & Dependencies"
echo ""

#stop/rm chain as dep doesn't work
eris services stop $SERVICE_NAME --all --data --force --rm --vol #--chain=$CHAIN_NAME 

#throws an error but cleans up anyway; see https://github.com/eris-ltd/eris-cli/issues/345
#TODO has been fixed!
echo ""
echo "API Error is false positive."
echo ""

eris chains stop $CHAIN_NAME --force --data --vol
eris chains rm $CHAIN_NAME --data 

echo "Removing latent dirs and files"
echo ""
rm -rf $CHAIN_DIR 
#rm -rf $HOME/.eris/keys/data/${ADDR}
rm $HOME/.eris/services/${SERVICE_NAME}".toml"
rm $HOME/.eris/chains/${CHAIN_NAME}".toml"

echo "Toadserver tests complete."
COMMENT
