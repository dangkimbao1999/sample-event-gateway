#!/bin/bash

# Default values
CONSUL_HOST="localhost"
CONSUL_PORT="8500"
DATA_ID="test-data"
NODE_ID="node1"
KV_PREFIX="streaming/data/"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --consul-host)
      CONSUL_HOST="$2"
      shift 2
      ;;
    --consul-port)
      CONSUL_PORT="$2"
      shift 2
      ;;
    --data-id)
      DATA_ID="$2"
      shift 2
      ;;
    --node-id)
      NODE_ID="$2"
      shift 2
      ;;
    --kv-prefix)
      KV_PREFIX="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Construct the Consul API URL
CONSUL_URL="http://${CONSUL_HOST}:${CONSUL_PORT}"
KV_PATH="${KV_PREFIX}${DATA_ID}"

# Set the data-to-node mapping in Consul
echo "Setting up data-to-node mapping in Consul..."
curl -X PUT "${CONSUL_URL}/v1/kv/${KV_PATH}" -d "${NODE_ID}"

if [ $? -eq 0 ]; then
  echo "Successfully set up mapping: ${DATA_ID} -> ${NODE_ID}"
else
  echo "Failed to set up mapping"
  exit 1
fi 