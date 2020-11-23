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

VENDOR='trava'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function normalize_release_type {
	case "${1}" in
	ea|*Experimental*) echo 'ea'
		;;
	*) echo 'ga'
		;;
	esac
}

function download {
	local tag_name="${1}"
	local asset_name="${2}"
	local filename="${asset_name}"

	local VERSION=""
	local ARCH="x86_64"
	local RELEASE_TYPE="ga"
	local OS=""
	local EXT=""

	# shellcheck disable=SC2016
	local tag_regex='s/^dcevm8u([0-9]+)b([0-9])+$/VERSION="8.0.$1+$2"/g'

	# Parse meta-data from version tag
	eval "$(perl -pe "${tag_regex}" <<< "${tag_name}")"

	# shellcheck disable=SC2016
	local filename_regex='s/^java8-openjdk-dcevm-(linux|osx|windows)\.(.*)$/OS="$1" EXT="$2"/g'

	# Parse meta-data from file name
	eval "$(perl -pe "${filename_regex}" <<< "${asset_name}")"

	local url="https://github.com/TravaOpenJDK/trava-jdk-8-dcevm/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${VENDOR}-${VERSION}-${OS}.${EXT}.json"
	local archive="${TEMP_DIR}/${VENDOR}-${VERSION}-${OS}.${EXT}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	else
		download_file "${url}" "${archive}" || return 1

		local json
		json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			"$(normalize_release_type "${RELEASE_TYPE}")" \
			"${VERSION}" \
			"${VERSION}" \
			'hotspot' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${EXT}" \
			'jdk' \
			'' \
			"${url}" \
			"$(hash_file 'md5' "${archive}" "${CHECKSUM_DIR}" "${asset_name}")" \
			"$(hash_file 'sha1' "${archive}" "${CHECKSUM_DIR}" "${asset_name}")" \
			"$(hash_file 'sha256' "${archive}" "${CHECKSUM_DIR}" "${asset_name}")" \
			"$(hash_file 'sha512' "${archive}" "${CHECKSUM_DIR}" "${asset_name}")" \
			"$(file_size "${archive}")" \
			"$(basename "${archive}")"
		)"

		echo "${json}" > "${metadata_file}"
		rm -f "${archive}"
	fi
}

RELEASE_FILE="${TEMP_DIR}/releases-${VENDOR}-8.json"
download_github_releases 'TravaOpenJDK' 'trava-jdk-8-dcevm' "${RELEASE_FILE}"

versions=$(jq -r '.[].tag_name' "${RELEASE_FILE}" | sort -V)
for version in ${versions}
do
	assets=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .assets[] | select(.content_type | startswith(\"application\")) | select(.name | contains(\"_source\") | not) | select(.name | endswith(\"jar\") | not) | .name" "${RELEASE_FILE}")
	for asset in ${assets}
	do
		download "${version}" "${asset}" || echo "Cannot download ${asset}"
	done
done

jq -s -S . "${METADATA_DIR}"/trava-8*.json > "${METADATA_DIR}/all.json"
