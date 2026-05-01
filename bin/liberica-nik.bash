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

VENDOR='liberica-nik'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

API_URL="https://api.bell-sw.com/v1/nik/releases"

function get_release_type {
	if [[ "${1}" = 'true' ]]
	then
		echo 'ga'
	else
		echo 'ea'
	fi
}

function normalize_liberica_arch {
	local arch="${1}"
	local bitness="${2}"
	if [[ "${arch}" == "x86" ]] && [[ "${bitness}" == "64" ]]
	then
		echo "x86_64"
	elif [[ "${arch}" == "arm" ]] && [[ "${bitness}" == "64" ]]
	then
		echo "aarch64"
	else
		echo "${arch}"
	fi
}

function normalize_features {
	local bundle_type="${1}"
	local os="${2}"
	declare -a features
	if [[ "${bundle_type}" == "full" ]]
	then
		features+=("libericafx" "minimal-vm" "javafx")
	fi
	if [[ "${os}" == "linux-musl" ]]
	then
		features+=("musl")
	fi
	echo "${features[@]}"
}

# Fetch releases
RELEASES_FILE="${TEMP_DIR}/releases.json"
curl -s "${API_URL}" > "${RELEASES_FILE}"

# Iterate over releases
# We filter for nik component
jq -c '.[] | select(.component == "nik")' "${RELEASES_FILE}" | while read -r release; do
	FILENAME=$(echo "${release}" | jq -r '.filename')
	URL=$(echo "${release}" | jq -r '.downloadUrl')
	VERSION=$(echo "${release}" | jq -r '.version')
	GA=$(echo "${release}" | jq -r '.GA')
	OS=$(echo "${release}" | jq -r '.os')
	ARCH=$(echo "${release}" | jq -r '.architecture')
	BITNESS=$(echo "${release}" | jq -r '.bitness')
	EXT=$(echo "${release}" | jq -r '.packageType')
	BUNDLE_TYPE=$(echo "${release}" | jq -r '.bundleType')

	# Extract Java version from components
	JAVA_VERSION=$(echo "${release}" | jq -r '.components[] | select(.component == "liberica") | .version')

	METADATA_FILE="${METADATA_DIR}/${FILENAME}.json"
	ARCHIVE="${TEMP_DIR}/${FILENAME}"

	if [[ -f "${METADATA_FILE}" ]]
	then
		echo "Skipping ${FILENAME}"
		continue
	fi

	echo "Processing ${FILENAME}"
	download_file "${URL}" "${ARCHIVE}" || continue

	# Generate metadata JSON
	json="$(metadata_json \
		"${VENDOR}" \
		"${FILENAME}" \
		"$(get_release_type "${GA}")" \
		"${VERSION}" \
		"${JAVA_VERSION}" \
		'graalvm' \
		"$(normalize_os "${OS}")" \
		"$(normalize_arch "$(normalize_liberica_arch "${ARCH}" "${BITNESS}")")" \
		"${EXT}" \
		'jdk' \
		"$(normalize_features "${BUNDLE_TYPE}" "${OS}")" \
		"${URL}" \
		"$(hash_file 'md5' "${ARCHIVE}" "${CHECKSUM_DIR}")" \
		"$(hash_file 'sha1' "${ARCHIVE}" "${CHECKSUM_DIR}")" \
		"$(hash_file 'sha256' "${ARCHIVE}" "${CHECKSUM_DIR}")" \
		"$(hash_file 'sha512' "${ARCHIVE}" "${CHECKSUM_DIR}")" \
		"$(file_size "${ARCHIVE}")" \
		"${FILENAME}"
	)"

	echo "${json}" > "${METADATA_FILE}"
	rm -f "${ARCHIVE}"
done

jq -s -S . "${METADATA_DIR}"/*.json > "${METADATA_DIR}/all.json"
