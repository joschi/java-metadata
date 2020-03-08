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

VENDOR='graalvm'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function download {
	local tag_name="${1}"
	local asset_name="${2}"
	local filename="${asset_name}"

	local url="https://github.com/oracle/graal/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	else
		local release_type
		local regex
		if echo "${asset_name}" | grep -q '^graalvm-ce-1.0.0-rc'
		then
			release_type="ea"
			# shellcheck disable=SC2016
			regex='s/^graalvm-ce-([0-9+.]{2,}-rc[0-9]+)-(linux|macos)-amd64\.(.+)$/OS="$2" VERSION="$1" EXT="$3"/g'
		else
			release_type="ga"
			# shellcheck disable=SC2016
			regex='s/^graalvm-ce-(linux|darwin|windows)-(aarch64|amd64)-([0-9+.]{2,}[^.]*)\.(.+)$/OS="$1" ARCH="$2" VERSION="$3" EXT="$4"/g'
		fi

		local OS=""
		local ARCH="amd64"
		local VERSION=""
		local EXT=""

		# Parse meta-data from file name
		eval "$(echo "${asset_name}" | perl -pe "${regex}")"

		download_file "${url}" "${archive}" || return 1

		local json
		json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			"${release_type}" \
			"${VERSION}" \
			'8' \
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
			"$(file_size "${archive}")"
		)"

		echo "${json}" > "${metadata_file}"
		rm -f "${archive}"
	fi
}

download_github_releases 'oracle' 'graal' "${TEMP_DIR}/releases-graalvm-legacy.json"

versions=$(jq -r '.[].tag_name | select(startswith("vm-"))' "${TEMP_DIR}/releases-graalvm-legacy.json" | sort -V)
for version in ${versions}
do
	assets=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .assets[].name | select(startswith(\"graalvm-ce\"))" "${TEMP_DIR}/releases-graalvm-legacy.json")
	for asset in ${assets}
	do
		download "${version}" "${asset}" || echo "Cannot download ${asset}"
	done
done
