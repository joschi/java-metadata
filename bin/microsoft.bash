#!/usr/bin/env bash
set -e
set -Euo pipefail

TEMP_DIR=$(mktemp -d)
trap 'rm -rf ${TEMP_DIR}' EXIT

if [[ "$#" -lt 2 ]]
then
	echo "Usage: ${0} metadata-directory checksum-directory"
	exit 1
fi

# shellcheck source=bin/functions.bash
source "$(dirname "${0}")/functions.bash"

VENDOR='microsoft'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

# shellcheck disable=SC2016
REGEX='s/^microsoft-jdk-([0-9+.]{3,})-(linux|macos|macOS|windows)-(x64|aarch64)\.(.*)$/VERSION="$1" OS="$2" ARCH="$3" ARCHIVE="$4"/g'

INDEX_FILE="${TEMP_DIR}/index.html"
download_file 'https://docs.microsoft.com/en-us/java/openjdk/download' "${INDEX_FILE}"

MSJDK_FILES=$(grep -o -E '<a href="https://aka.ms/download-jdk/(microsoft-jdk-.+-(linux|macos|macOS|windows)-(x64|aarch64)\.(tar\.gz|zip|msi|dmg|pkg))"' "${INDEX_FILE}" | perl -pe 's/<a href="https:\/\/aka.ms\/download-jdk\/(.+)"/$1/g' | sort -V)
for MSJDK_FILE in ${MSJDK_FILES}
do
	METADATA_FILE="${METADATA_DIR}/${MSJDK_FILE}.json"
	MSJDK_ARCHIVE="${TEMP_DIR}/${MSJDK_FILE}"
	MSJDK_URL="https://aka.ms/download-jdk/${MSJDK_FILE}"
	if [[ -f "${METADATA_FILE}" ]]
	then
		echo "Skipping ${MSJDK_FILE}"
	else
		download_file "${MSJDK_URL}" "${MSJDK_ARCHIVE}"
		VERSION=""
		OS=""
		ARCH=""
		ARCHIVE=""

		# Parse meta-data from file name
		PARSED_NAME=$(perl -pe "${REGEX}" <<< "${MSJDK_FILE}")
		if [[ "${PARSED_NAME}" = "${MSJDK_FILE}" ]]
		then
			echo "Regular expression didn't match ${MSJDK_FILE}"
			continue
		else
			eval "${PARSED_NAME}"
		fi
		if [[ "$ARCH" = "aarch64" ]]
		then
			RELEASE_TYPE="ea"
		else
			RELEASE_TYPE="ga"
		fi

		METADATA_JSON="$(metadata_json \
			"${VENDOR}" \
			"${MSJDK_FILE}" \
			"$(normalize_release_type "${RELEASE_TYPE}")" \
			"${VERSION}" \
			"${VERSION}" \
			'hotspot' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${ARCHIVE}" \
			"jdk" \
			"" \
			"${MSJDK_URL}" \
			"$(hash_file 'md5' "${MSJDK_ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha1' "${MSJDK_ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha256' "${MSJDK_ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha512' "${MSJDK_ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(file_size "${MSJDK_ARCHIVE}")" \
			"${MSJDK_FILE}"
		)"

		echo "${METADATA_JSON}" > "${METADATA_FILE}"
		rm -f "${MSJDK_ARCHIVE}"
	fi
done

jq -s -S . "${METADATA_DIR}"/microsoft-jdk-*.json > "${METADATA_DIR}/all.json"
