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

VENDOR='liberica'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function get_release_type {
	if [[ "${2}" = 'true' ]]
	then
		echo 'ea'
	else
		case "${1}" in
		*ea*) echo 'ea'
			;;
		*) echo 'ga'
			;;
		esac
	fi
}

function normalize_features {
	declare -a features
	if [[ "${1}" == "lite" ]] || [[ "${1}" == "musl-lite" ]]
	then
		features+=("lite")
	fi
	if [[ "${1}" == "full" ]]
	then
		features+=("libericafx" "minimal-vm" "javafx")
	fi
	if [[ "${1}" == "fx" ]]
	then
		features+=("javafx")
	fi
	if [[ "${1}" == "musl" ]] || [[ "${1}" == "musl-lite" ]]
	then
		features+=("musl")
	fi
	echo "${features[@]}"
}

function download {
	local tag_name="${1}"
	local asset_name="${2}"
	local is_prerelease="${3}"
	local filename="${asset_name}"

	local url="https://github.com/bell-sw/Liberica/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${tag_name} - ${filename}"
	else
		# shellcheck disable=SC2016
		local regex='s/^bellsoft-(jre|jdk)(.+)-(linux|windows|macos|solaris)-(amd64|i386|i586|aarch64|arm64|ppc64le|arm32-vfp-hflt|x64|sparcv9)-?(fx|lite|full|musl|musl-lite)?\.(deb|rpm|msi|dmg|pkg|tar\.gz|zip)$/IMAGE_TYPE="$1" VERSION="$2" OS="$3" ARCH="$4" FEATURES="$5" EXT="$6"/g'

		local IMAGE_TYPE=""
		local VERSION=""
		local OS=""
		local ARCH=""
		local FEATURES=""
		local EXT=""

		# Parse meta-data from file name
		eval "$(echo "${asset_name}" | perl -pe "${regex}")"

		download_file "${url}" "${archive}" || return 1

		local json
		json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			"$(get_release_type "${VERSION}" "${is_prerelease}")" \
			"${VERSION}" \
			"${VERSION}" \
			'hotspot' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${EXT}" \
			"${IMAGE_TYPE}" \
			"$(normalize_features "${FEATURES}")" \
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

download_github_releases 'bell-sw' 'Liberica' "${TEMP_DIR}/releases-${VENDOR}.json"

versions=$(jq -r '.[].tag_name' "${TEMP_DIR}/releases-${VENDOR}.json" | sort -V)
for version in ${versions}
do
	prerelease=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .prerelease" "${TEMP_DIR}/releases-${VENDOR}.json")
	assets=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .assets[] | select(.content_type != \"text/plain\") | select (.name | endswith(\".bom\") | not) | select (.name | endswith(\"-src.tar.gz\") | not) | select (.name | endswith(\"-src-full.tar.gz\") | not) | select (.name | contains(\"-full-nosign\") | not) | .name" "${TEMP_DIR}/releases-${VENDOR}.json")
	for asset in ${assets}
	do
		download "${version}" "${asset}" "${prerelease}" || echo "Cannot download ${asset}"
	done
done

jq -s -S . "${METADATA_DIR}"/bellsoft-*.json > "${METADATA_DIR}/all.json"
