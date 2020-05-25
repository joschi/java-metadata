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

VENDOR='adoptopenjdk'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function normalize_version {
	local jvm_impl="$1"
	local name="$2"
	local version="$3"

	if [[ "${jvm_impl}" == "openj9" ]] && [[ "${name}" =~ "openj9" ]] && [[ ! "${version}" =~ "openj9" ]]
	then
		local openj9_version
		openj9_version=$(echo "${name}" | perl -pe 's/^.*_(openj9-\d+\.\d+\.\d+)\..+$/$1/')
		echo "${version}.${openj9_version}"
	else
		echo "${version}"
	fi
}

function normalize_features {
	declare -a features
	if [[ "${1}" == "large" ]]
	then
		features+=("large_heap")
	fi
	# Handle miscategorized builds: https://github.com/AdoptOpenJDK/openjdk-api-v3/issues/204
	if [[ "${2}" =~ 'LH' ]]
	then
		features+=("large_heap")
	fi
	echo "${features[@]}"
}

function download {
	local json
	json=$(echo "$1" | base64 -d)
	local filename
	filename=$(jq -r '.name' <<< "${json}")

	local image_type
	image_type="$(jq -r '.image_type' <<< "${json}")"

	if [[ "${image_type}" = 'testimage' ]]
	then
		echo "Skipping test image ${filename}"
		return 0
	fi

	local ext
	# shellcheck disable=SC2016
	ext=$(echo "${filename}" | perl -pe 's/^.*\.(zip|tar\.gz)$/$1/g')
	local url
	url=$(jq -r '.link' <<< "${json}")
	local archive="${METADATA_DIR}/${filename}"

	local version
	version="$(jq -r '.version' <<< "${json}")"
	local jvm_impl
	jvm_impl="$(jq -r '.jvm_impl' <<< "${json}")"
	local normalized_version
	normalized_version="$(normalize_version "${jvm_impl}" "${filename}" "${version}")"

	local metadata_file="${METADATA_DIR}/${filename}.json"
	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	else
		if ! download_file "${url}" "${archive}"
		then
			echo "Failed to download ${url}"
			return 0
		fi

		local md_json
		md_json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			'ga' \
			"${normalized_version}" \
			"$(jq -r '.java_version' <<< "${json}")" \
			"${jvm_impl}" \
			"$(normalize_os "$(jq -r '.os' <<< "${json}")")" \
			"$(normalize_arch "$(jq -r '.architecture' <<< "${json}")")" \
			"${ext}" \
			"$(jq -r '.image_type' <<< "${json}")" \
			"$(normalize_features "$(jq -r '.heap_size' <<< "${json}")" "${filename}")" \
			"${url}" \
			"$(hash_file 'md5' "${archive}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha1' "${archive}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha256' "${archive}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha512' "${archive}" "${CHECKSUM_DIR}")" \
			"$(file_size "${archive}")"
		)"

		echo "${md_json}" > "${metadata_file}"
		rm -f "${METADATA_DIR}/${filename}"
	fi
}

RELEASES_FILE="${TEMP_DIR}/available-releases.json"
download_file 'https://api.adoptopenjdk.net/v3/info/available_releases' "${RELEASES_FILE}"
AVAILABLE_RELEASES=$(jq '.available_releases[]' "${RELEASES_FILE}")

for release in ${AVAILABLE_RELEASES}
do
	page=0
	while download_file "https://api.adoptopenjdk.net/v3/assets/feature_releases/${release}/ga?page=${page}&page_size=20&project=jdk&sort_order=ASC&vendor=adoptopenjdk" "${TEMP_DIR}/release-${release}-${page}.json"
	do
		page=$((page+1))
	done
done

FLATTEN_QUERY='add |
.[] |
[{
	release_type: .release_type,
	java_version: .version_data.openjdk_version,
	version: .version_data.semver,
	binary: .binaries[],
}] |
.[] |
{
	release_type,
	java_version,
	version,
	architecture: .binary.architecture,
	os: .binary.os,
	heap_size: .binary.heap_size,
	image_type: .binary.image_type,
	jvm_impl: .binary.jvm_impl,
	link: .binary.package.link,
	name: .binary.package.name
} | @base64'

for json_b64 in $(jq -r -s "${FLATTEN_QUERY}" "${TEMP_DIR}"/release-*.json)
do
	download "${json_b64}"
done

jq -s -S . "${METADATA_DIR}"/OpenJDK*.json > "${METADATA_DIR}/all.json"
