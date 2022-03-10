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
if [ -z ${LOG_LEVEL} ]; then
    export LOG_LEVEL=error
fi

if [ -z ${DICE_IS_EDGE} ]; then
    echo $DICE_IS_EDGE
fi
if [ -z ${COLLECTOR_URL} ]; then
  if [ $DICE_IS_EDGE == 'true' ]; then
    export COLLECTOR_URL=$COLLECTOR_PUBLIC_UR
  else
    export COLLECTOR_URL='http://'$COLLECTOR_ADDR
  fi
fi

if [ -z ${MASTER_VIP_URL} ]; then
    export MASTER_VIP_URL='https://kubernetes.default.svc:443'
fi

if [ -z ${CONFIG_FILE} ]; then
  CONFIG_FILE=/fluent-bit/etc/ds/fluent-bit.conf
fi

if [ -z ${COLLECTOR_URL} ]; then
  echo "env COLLECTOR_URL or COLLECTOR_PUBLIC_UR or COLLECTOR_ADDR unset!"
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
export COLLECTOR_PORT=$port
export COLLECTOR_HOST=$host

if [ "$CONFIG_FILE" == "/fluent-bit/etc/ds/fluent-bit.conf" ]; then
   credential_file='/erda-cluster-credential/CLUSTER_ACCESS_KEY'
   if [ -z ${CLUSTER_ACCESS_KEY} ]; then
   if [ -e "$credential_file" ]; then
     export CLUSTER_ACCESS_KEY=$(cat $credential_file)
   else
     echo "$credential_file must existed or specify env CLUSTER_ACCESS_KEY"
     exit 1
   fi
   fi
fi


echo 'LOG_LEVEL: '$LOG_LEVEL
echo 'COLLECTOR_PORT: '$COLLECTOR_PORT
echo 'COLLECTOR_HOST: '$COLLECTOR_HOST

# --- init work block ---

/fluent-bit/bin/fluent-bit -c $CONFIG_FILE &

child=$!
wait "$child"