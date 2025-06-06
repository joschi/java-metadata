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

VENDOR='kona'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function download {
	local tag_name="${1}"
	local asset_name="${2}"
	local filename="${asset_name}"

	local url="https://github.com/Tencent/TencentKona-21/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	else
		download_file "${url}" "${archive}" || return 1

		local VERSION=""
		local JAVA_VERSION=""
		local RELEASE_TYPE=""
		local OS=""
		local ARCH=""
		local EXT=""
		local FEATURES=""

		# shellcheck disable=SC2016
		local regex='s/^TencentKona-([0-9b.]{1,})(?:[_-](ea))?[-_]jdk_(linux|macosx|windows)-(aarch64|x86_64)(?:_(notarized|signed))?\.(tar\.gz|zip)$/VERSION="$1" RELEASE_TYPE="$2" OS="$3" ARCH="$4" JAVA_VERSION="$1" FEATURES="$5" EXT="$6"/g'

		# Parse meta-data from file name
		eval "$(perl -pe "${regex}" <<< "${asset_name}")"

		if [[ -z "${RELEASE_TYPE}" ]]
		then
			RELEASE_TYPE="ga"
		fi

		if [[ -z "${VERSION}" ]]
		then
			echo "Unable to parse ${asset_name}"
			return 1
		fi

		local json
		json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			"$(normalize_release_type "${RELEASE_TYPE}")" \
			"${VERSION}" \
			"${JAVA_VERSION}" \
			'hotspot' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${EXT}" \
			'jdk' \
			"${FEATURES}" \
			"${url}" \
			"$(hash_file 'md5' "${archive}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha1' "${archive}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha256' "${archive}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha512' "${archive}" "${CHECKSUM_DIR}")" \
			"$(file_size "${archive}")" \
			"${filename}"
		)"

		echo "${json}" > "${metadata_file}"
		rm -f "${archive}"
	fi
}

RELEASE_FILE="${TEMP_DIR}/releases-${VENDOR}-21.json"
download_github_releases 'Tencent' 'TencentKona-21' "${RELEASE_FILE}"

versions=$(jq -r '.[].tag_name' "${RELEASE_FILE}" | sort -V)
for version in ${versions}
do
	assets=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .assets[] | select(.content_type | startswith(\"application\")) | select(.name | contains(\"_source\") | not) | select(.name | endswith(\"md5\") | not) | .name" "${RELEASE_FILE}")
	for asset in ${assets}
	do
		download "${version}" "${asset}" || echo "Cannot download ${asset}"
	done
done

jq -s -S . "${METADATA_DIR}"/Tencent*.json > "${METADATA_DIR}/all.json"
