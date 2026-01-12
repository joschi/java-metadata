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

VENDOR='semeru'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function normalize_release_type {
	case "${1}" in
	*) echo 'ga'
		;;
	esac
}

function download {
	local tag_name="${1}"
	local asset_name="${2}"
	local filename="${asset_name}"

	local url="https://github.com/ibmruntimes/semeru16-binaries/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	else
		download_file "${url}" "${archive}" || return 1

		local JAVA_VERSION=""
		local OPENJ9_VERSION=""
		#local RPM_VERSION=""
		local RELEASE_TYPE="ga"
		local IMAGE_TYPE=""
		local OS=""
		local ARCH=""
		local EXT=""
		local regex

		# shellcheck disable=SC2016
		version_regex='s/^jdk-(.*)_openj9-(.*)$/JAVA_VERSION="$1" OPENJ9_VERSION="$2"/g'
		eval "$(perl -pe "${version_regex}" <<< "${tag_name}")"
		local VERSION="${JAVA_VERSION}_openj9-${OPENJ9_VERSION}"

		if [[ "${filename}" =~ \.rpm$ ]]
		then
			OS='linux'
			EXT='rpm'

			# shellcheck disable=SC2016
			regex='s/^ibm-semeru-open-[0-9]+-(jre|jdk)-(.+)\.(x86_64|s390x|ppc64|ppc64le|aarch64)\.rpm$/IMAGE_TYPE="$1" RPM_VERSION="$2" ARCH="$3"/g'
		else
			# shellcheck disable=SC2016
			regex='s/^ibm-semeru-open-(jre|jdk)_(x64|x86-32|s390x|ppc64|ppc64le|aarch64)_(aix|linux|mac|windows)_.+_openj9-.+\.(tar\.gz|zip|msi)$/IMAGE_TYPE="$1" ARCH="$2" OS="$3" EXT="$4"/g'
		fi

		# Parse meta-data from file name
		eval "$(perl -pe "${regex}" <<< "${asset_name}")"

		local json
		json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			"$(normalize_release_type "${RELEASE_TYPE}")" \
			"${VERSION}" \
			"${JAVA_VERSION}" \
			'openj9' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${EXT}" \
			"${IMAGE_TYPE}" \
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

RELEASE_FILE="${TEMP_DIR}/releases-${VENDOR}-16.json"
download_github_releases 'ibmruntimes' 'semeru16-binaries' "${RELEASE_FILE}"

versions=$(jq -r '.[].tag_name' "${RELEASE_FILE}" | sort -V)
for version in ${versions}
do
	assets=$(jq -r  ".[] | select(.prerelease == false) | select(.tag_name == \"${version}\") | .assets[] | select(.name | endswith(\"zip\") or endswith(\"tar.gz\") or endswith(\"msi\") or endswith(\"rpm\")) | select(.name | contains(\"debugimage\") | not) | select(.name | contains(\"testimage\") | not) | select(.name | contains(\"certified\") | not) | .name" "${RELEASE_FILE}")
	for asset in ${assets}
	do
		download "${version}" "${asset}" || echo "Cannot download ${asset}"
	done
done

jq -s -S . "${METADATA_DIR}"/ibm-semeru-*.json > "${METADATA_DIR}/all.json"
