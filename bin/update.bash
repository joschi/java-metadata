#!/usr/bin/env bash
#set -x
set -e
set -Euo pipefail

TEMP_DIR=$(mktemp -d)
trap 'rm -rf ${TEMP_DIR}' EXIT

SCRIPT_DIR=$(dirname "${0}")

# shellcheck source=bin/functions.bash
source "${SCRIPT_DIR}/functions.bash"

ROOT_DIR='./docs'
ensure_directory "${ROOT_DIR}/checksums"
ensure_directory "${ROOT_DIR}/metadata"
 
CHECKSUM_DIR=$(readlink -f "${ROOT_DIR}/checksums")
METADATA_DIR=$(readlink -f "${ROOT_DIR}/metadata")

function cmd() {
	echo "${SCRIPT_DIR}/${1}.bash"
}

vendors=(
	"$(cmd 'adoptopenjdk')"
	"$(cmd 'temurin')"
	"$(cmd 'corretto')"
	"$(cmd 'semeru8')"
	"$(cmd 'semeru11')"
	"$(cmd 'semeru16')"
	"$(cmd 'semeru17')"
	"$(cmd 'graalvm-legacy')"
	"$(cmd 'graalvm-ce')"
	"$(cmd 'graalvm-community')"
	"$(cmd 'zulu')"
	"$(cmd 'sapmachine')"
	"$(cmd 'liberica')"
	"$(cmd 'dragonwell8')"
	"$(cmd 'dragonwell11')"
	"$(cmd 'dragonwell17')"
	"$(cmd 'oracle')"
	"$(cmd 'oracle-graalvm')"
	"$(cmd 'openjdk')"
	"$(cmd 'openjdk-panama')"
	"$(cmd 'openjdk-valhalla')"
	"$(cmd 'java-se-ri')"
	"$(cmd 'mandrel')"
	"$(cmd 'trava8')"
	"$(cmd 'trava11')"
	"$(cmd 'microsoft')"
	"$(cmd 'kona8')"
	"$(cmd 'kona11')"
	"$(cmd 'kona17')"
)

printf '%s\n' "${vendors[@]}" | parallel -P 4 --verbose "bash {} ${METADATA_DIR}/vendor ${CHECKSUM_DIR} ; echo \"EXIT CODE: \$?\""

jq -s 'add' "${METADATA_DIR}"/vendor/*/all.json > "${METADATA_DIR}/all.json"
aggregate_metadata "${METADATA_DIR}/all.json" "${METADATA_DIR}"
