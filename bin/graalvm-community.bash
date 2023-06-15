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

VENDOR='graalvm-community'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function download {
	local tag_name="${1}"
	local asset_name="${2}"
	local filename="${asset_name}"

	local url="https://github.com/graalvm/graalvm-ce-builds/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	else
		# Starting from graalvm 23 : graalvm-community-jdk-17.0.7_macos-aarch64_bin.tar.gz
		#                            graalvm-community-jdk-17.0.7_linux-x64_bin.tar.gz
		# shellcheck disable=SC2016
		local regex='s/^graalvm-community-jdk-([0-9]{1,2}\.[0-9]{1}\.[0-9]{1,3})_(linux|macos|windows)-(aarch64|x64)_bin\.(zip|tar\.gz)$/JAVA_VERSION="$1" OS="$2" ARCH="$3" EXT="$4"/g'

		local JAVA_VERSION=""
		local OS=""
		local ARCH=""
		local EXT=""

		# Parse meta-data from file name
		eval "$(echo "${asset_name}" | perl -pe "${regex}")"

		download_file "${url}" "${archive}" || return 1

		local json
		json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			'ga' \
			"${JAVA_VERSION}" \
			"${JAVA_VERSION}" \
			'graalvm' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${EXT}" \
			'jdk' \
			'' \
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

download_github_releases 'graalvm' 'graalvm-ce-builds' "${TEMP_DIR}/releases-graalvm-community.json"

versions=$(jq -r '.[].tag_name' "${TEMP_DIR}/releases-graalvm-community.json" | sort -V | grep "^jdk")
for version in ${versions}
do
	assets=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .assets[].name | select(startswith(\"graalvm-community\")) | select(endswith(\"tar.gz\") or endswith(\"zip\"))" "${TEMP_DIR}/releases-graalvm-community.json")
	for asset in ${assets}
	do
		download "${version}" "${asset}" || echo "Cannot download ${asset}"
	done
done

jq -s -S . "${METADATA_DIR}"/graalvm-community-*.json > "${METADATA_DIR}/all.json"
