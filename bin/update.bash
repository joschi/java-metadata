#!/usr/bin/env bash
#set -x
set -e
set -Euo pipefail

TEMP_DIR=$(mktemp -d)
trap 'rm -rf ${TEMP_DIR}' EXIT

# shellcheck source=bin/functions.bash
source "$(dirname "${0}")/functions.bash"

ROOT_DIR='./docs'
ensure_directory "${ROOT_DIR}/checksums"
ensure_directory "${ROOT_DIR}/metadata"
 
CHECKSUM_DIR=$(readlink -f "${ROOT_DIR}/checksums")
METADATA_DIR=$(readlink -f "${ROOT_DIR}/metadata")

for vendor in \
	'adoptopenjdk' \
	'corretto' \
	'graalvm-legacy' \
	'graalvm' \
	'zulu' \
	'sapmachine' \
	'liberica' \
	'dragonwell8' \
	'dragonwell11' \
	'openjdk' \
	'openjdk-loom' \
	'openjdk-panama' \
	'java-se-ri' \
	'mandrel' \
	'trava8' \
	'trava11' \
	'microsoft'
do
	cmd="$(dirname "${0}")/${vendor}.bash"
	if [[ -x "${cmd}" ]]
	then
		"${cmd}" "${METADATA_DIR}/vendor" "${CHECKSUM_DIR}"
	else
		echo "Unable to update metadata for vendor ${vendor}"
	fi
done

jq -s 'add' "${METADATA_DIR}"/vendor/*/all.json > "${METADATA_DIR}/all.json"
aggregate_metadata "${METADATA_DIR}/all.json" "${METADATA_DIR}"
