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

VENDOR='corretto'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function get_extensions {
	case "${1}" in
	"linux") echo 'tar.gz'
		;;
	"macosx") echo 'tar.gz' 'pkg'
		;;
	"windows") echo 'zip' 'msi'
		;;
	esac
}

function archive_filename {
	local version="${1}"
	local os="${2}"
	local arch="${3}"
	local ext="${4}"
	local image_type="${5}"
	if [[ "${IMAGE_TYPE}" = 'none' ]]
	then
		echo "amazon-corretto-${version}-${os}-${arch}.${ext}"
	else
		echo "amazon-corretto-${version}-${os}-${arch}-${image_type}.${ext}"
	fi
}

function get_archs_for_os {
	case "${1}" in
	'linux') echo 'x64' 'aarch64'
		;;
	'alpine-linux') echo 'x64'
		;;
	'macosx') echo 'x64'
		;;
	'windows') echo 'x64' 'x86'
		;;
	esac
}

function get_exts_for_os {
	case "${1}" in
	'linux') echo 'tar.gz' 'rpm' 'deb'
		;;
	'alpine-linux') echo 'tar.gz'
		;;
	'macosx') echo 'tar.gz' 'pkg'
		;;
	'windows') echo 'zip' 'msi'
		;;
	esac
}

function get_image_types_for_os_and_ext {
	case "${1}" in
	'linux') echo 'none'
		;;
	'alpine-linux') echo 'none'
		;;
	'macosx') echo 'none'
		;;
	'windows')
		if [[ "$2" = 'zip' ]]
		then
			echo 'jre' 'jdk'
		else
			echo 'none'
		fi
		;;
	esac
}

function normalize_features {
	case "${1}" in
	'linux'|'macosx'|'windows')
		;;
	'alpine-linux') echo 'musl'
		;;
	esac
}
function download {
	local version="${1}"
	local os="${2}"
	local arch="${3}"
	local ext="${4}"
	local image_type="${5}"
	local filename
	filename="$(archive_filename "${version}" "${os}" "${arch}" "${ext}" "${image_type}")"

	local url
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	else
		if check_url_exists "https://corretto.aws/downloads/resources/${version}/${filename}"
		then
			url="https://corretto.aws/downloads/resources/${version}/${filename}"
		elif check_url_exists "https://d3pxv6yz143wms.cloudfront.net/${version}/${filename}"
		then
			url="https://d3pxv6yz143wms.cloudfront.net/${version}/${filename}"
		else
			echo "Couldn't find download URL for ${filename}"
			return 1
		fi

		download_file "${url}" "${archive}" || return 1

		if [[ "${image_type}" = "none" ]]
		then
			image_type="jdk"
		fi

		local json
		json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			'ga' \
			"${version}" \
			"${version}" \
			'hotspot' \
			"$(normalize_os "${os}")" \
			"$(normalize_arch "${arch}")" \
			"${ext}" \
			"${image_type}" \
			"$(normalize_features "${os}")" \
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

download_github_releases 'corretto' 'corretto-8' "${TEMP_DIR}/releases-corretto-8.json"
download_github_releases 'corretto' 'corretto-11' "${TEMP_DIR}/releases-corretto-11.json"
download_github_releases 'corretto' 'corretto-jdk' "${TEMP_DIR}/releases-corretto-jdk.json"

jq -s 'add' "${TEMP_DIR}/releases-corretto-8.json" "${TEMP_DIR}/releases-corretto-11.json" "${TEMP_DIR}/releases-corretto-jdk.json" > "${TEMP_DIR}/releases-corretto.json"

for CORRETTO_VERSION in $(jq -r '.[].tag_name' "${TEMP_DIR}/releases-corretto.json" | sort -V)
do
	for OS in 'linux' 'macosx' 'windows' 'alpine-linux'
	do
		for ARCH in $(get_archs_for_os "${OS}")
		do
			for EXT in $(get_exts_for_os "${OS}")
			do
				for IMAGE_TYPE in $(get_image_types_for_os_and_ext "${OS}" "${EXT}")
				do
					download "${CORRETTO_VERSION}" "${OS}" "${ARCH}" "${EXT}" "${IMAGE_TYPE}" || echo "Cannot download $(archive_filename "${CORRETTO_VERSION}" "${OS}" "${ARCH}" "${EXT}" "${IMAGE_TYPE}")"
				done
			done
		done
	done
done

jq -s -S . "${METADATA_DIR}"/amazon-corretto-*.json > "${METADATA_DIR}/all.json"
