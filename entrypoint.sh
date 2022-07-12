#!/bin/bash

_term() {
  echo "Caught SIGTERM signal!"
  kill -TERM "$child" 2>/dev/null
}

_int() {
  echo "Caught SIGINT signal!"
  kill -INT "$child" 2>/dev/null
}

trap _term SIGTERM
trap _int SIGINT

# --- init work block ---


if [ -z ${FLUENTBIT_INPUT_TAIL_EXCLUDE_PATH} ]; then
    export FLUENTBIT_INPUT_TAIL_EXCLUDE_PATH='/var/log/containers/*fluent-bit*.log'
fi

if [ -z ${LOG_LEVEL} ]; then
    export LOG_LEVEL=error
fi

if [ -z ${DICE_IS_EDGE} ]; then
    export DICE_IS_EDGE='false'
fi

if [ -z ${COLLECTOR_URL} ]; then
  if [ $DICE_IS_EDGE == 'true' ]; then
    export COLLECTOR_URL=$COLLECTOR_PUBLIC_URL
  else
    export COLLECTOR_URL='http://'$COLLECTOR_ADDR
  fi
fi

if [ -z ${MASTER_VIP_URL} ]; then
    export MASTER_VIP_URL='https://kubernetes.default.svc:443'
fi

if [ -z ${FLUENTBIT_THROTTLE_RATE} ]; then
    export FLUENTBIT_THROTTLE_RATE=1000
fi
if [ -z ${FLUENTBIT_THROTTLE_RETAIN} ]; then
    export FLUENTBIT_THROTTLE_RETAIN=true
fi
if [ -z ${FLUENTBIT_THROTTLE_PRINT_STATUS} ]; then
    export FLUENTBIT_THROTTLE_PRINT_STATUS=false
fi

if [ -z ${CONFIG_FILE} ]; then
    export CONFIG_FILE=/fluent-bit/etc/ds/fluent-bit.conf
fi

# select runtime's specific config
if [ -z ${DICE_CONTAINER_RUNTIME} ]; then
    export DICE_CONTAINER_RUNTIME=docker
fi
# work around issue: https://github.com/fluent/fluent-bit/issues/2020
if [ "$DICE_CONTAINER_RUNTIME" == docker ]; then
    sed -i -- 's/${INCLUDE_RUNTIME_CONF}/docker-runtime.conf/g' $CONFIG_FILE
elif [ "$DICE_CONTAINER_RUNTIME" == containerd ]; then
    sed -i -- 's/${INCLUDE_RUNTIME_CONF}/cri-runtime.conf/g' $CONFIG_FILE
else
    echo "invaild DICE_CONTAINER_RUNTIME=$DICE_CONTAINER_RUNTIME"
    exit 1
fi


if [ -z ${COLLECTOR_URL} ]; then
  echo "env COLLECTOR_URL or COLLECTOR_PUBLIC_URL or COLLECTOR_ADDR unset!"
  exit 1
fi

# extract the protocol
proto="$(echo $COLLECTOR_URL | grep :// | sed -e's,^\(.*://\).*,\1,g')"

# remove the protocol -- updated
url=$(echo $COLLECTOR_URL | sed -e s,$proto,,g)

# extract the user (if any)
#user="$(echo $url | grep @ | cut -d@ -f1)"

# extract the host and port -- updated
hostport=$(echo $url | sed -e s,$user@,,g | cut -d/ -f1)

# by request host without port
host="$(echo $hostport | sed -e 's,:.*,,g')"
# by request - try to extract the port
port="$(echo $hostport | sed -e 's,^.*:,:,g' -e 's,.*:\([0-9]*\).*,\1,g' -e 's,[^0-9],,g')"

# extract the path (if any)
#path="$(echo $url | grep / | cut -d/ -f2-)"

if [ -z ${port} ]; then
  if [ $proto == 'http://' ]; then
    port=80
  elif [ $proto == 'https://' ]; then
    port=443
  else
    port='unknown'
  fi
fi

# tls config
if [ -z ${OUTPUT_HTTP_TLS} ]; then
  if [ $proto == 'https://' ]; then
    export OUTPUT_HTTP_TLS='On'
  else
    export OUTPUT_HTTP_TLS='Off'
  fi
fi

export COLLECTOR_PORT=$port
export COLLECTOR_HOST=$host

# TODO. use basic auth temporarily
#if [ "$CONFIG_FILE" == "/fluent-bit/etc/ds/fluent-bit.conf" ]; then
#   credential_file='/erda-cluster-credential/CLUSTER_ACCESS_KEY'
#   if [ -z ${CLUSTER_ACCESS_KEY} ]; then
#   if [ -e "$credential_file" ]; then
#     export CLUSTER_ACCESS_KEY=$(cat $credential_file)
#   else
#     echo "$credential_file must existed or specify env CLUSTER_ACCESS_KEY"
#     exit 1
#   fi
#   fi
#fi


echo 'LOG_LEVEL: '$LOG_LEVEL
echo 'COLLECTOR_PORT: '$COLLECTOR_PORT
echo 'COLLECTOR_HOST: '$COLLECTOR_HOST
echo 'OUTPUT_HTTP_TLS: '$OUTPUT_HTTP_TLS
echo "CONFIG_FILE: "$CONFIG_FILE

# --- init work block ---

/fluent-bit/bin/fluent-bit -c $CONFIG_FILE &

child=$!
wait "$child"