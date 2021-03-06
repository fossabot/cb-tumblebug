#!/bin/bash

#function list_spec() {
    FILE=../conf.env
    if [ ! -f "$FILE" ]; then
        echo "$FILE does not exist."
        exit
    fi

    source ../conf.env
    AUTH="Authorization: Basic $(echo -n $ApiUsername:$ApiPassword | base64)"

    echo "####################################################################"
    echo "## 7. spec: List"
    echo "####################################################################"


    $CBTUMBLEBUG_ROOT/src/api/grpc/cbadm/cbadm spec list --config $CBTUMBLEBUG_ROOT/src/api/grpc/cbadm/grpc_conf.yaml -o json --ns $NS_ID

#}

#list_spec