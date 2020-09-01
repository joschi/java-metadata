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

VENDOR='java-se-ri'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function normalize_release_type {
	case "${1}" in
	*-ea*) echo 'ea'
		;;
	*) echo 'ga'
		;;
	esac
}

# shellcheck disable=SC2016
REGEX='s/^openjdk-([0-9ub-]{1,}[^_]*)[-_](linux|osx|windows)-(aarch64|x64-musl|x64|i586).*\.(tar\.gz|zip)$/VERSION="$1" OS="$2" ARCH="$3" EXT="$4"/g'

for URL_VERSION in '7' '8-MR3' '9' '10' '11' '12' '13' '14'
do
	download_file "http://jdk.java.net/java-se-ri/${URL_VERSION}" "${TEMP_DIR}/index-${URL_VERSION}.html"
done

URLS=$(grep -h -o -E 'href="https://download.java.net/.*/openjdk-[^/]*\.(tar\.gz|zip)"' "${TEMP_DIR}"/index-*.html | grep -v '[-_]src' | perl -pe 's/href="(.+)"/$1/g' | sort -V)
for URL in ${URLS}
do
	FILE="$(perl -pe 's/https.*\/([^\/]+)/$1/g' <<< "${URL}")"
	METADATA_FILE="${METADATA_DIR}/${FILE}.json"
	ARCHIVE="${TEMP_DIR}/${FILE}"
	if [[ -f "${METADATA_FILE}" ]]
	then
		echo "Skipping ${FILE}"
	else
		download_file "${URL}" "${ARCHIVE}"
		VERSION=""
		OS=""
		ARCH=""
		EXT=""

		# Parse meta-data from file name
		eval "$(perl -pe "${REGEX}" <<< "${FILE}")"

		FEATURES=""
		if [[ "${ARCH}" =~ "x64-musl" ]]
		then
			ARCH="x64"
			FEATURES="musl"
		fi

		METADATA_JSON="$(metadata_json \
			"${VENDOR}" \
			"${FILE}" \
			"$(normalize_release_type "${VERSION}")" \
			"${VERSION}" \
			"${VERSION}" \
			'hotspot' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${EXT}" \
			"jdk" \
			"${FEATURES}" \
			"${URL}" \
			"$(hash_file 'md5' "${ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha1' "${ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha256' "${ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha512' "${ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(file_size "${ARCHIVE}")"
		)"

		echo "${METADATA_JSON}" > "${METADATA_FILE}"
		rm -f "${ARCHIVE}"
	fi
done

jq -s -S . "${METADATA_DIR}"/openjdk*.json > "${METADATA_DIR}/all.json"
