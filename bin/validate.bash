#!/usr/bin/env bash
set -e
set -Euo pipefail

METADATA_FILE="$1"

if [[ ! -r "${METADATA_FILE}" ]]
then
	echo "Cannot read input file ${METADATA_FILE}" 1>&2
	exit 1
fi

URL=$(jq -r '.url' "${METADATA_FILE}")
curl --fail --silent --head -X GET -L "${URL}" > /dev/null || echo "${METADATA_FILE}"
