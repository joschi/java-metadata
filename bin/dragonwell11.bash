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

VENDOR='dragonwell'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

function normalize_release_type {
	case "${1}" in
	ea|*Experimental*|FP1) echo 'ea'
		;;
	*) echo 'ga'
		;;
	esac
}

function download {
	local tag_name="${1}"
	local asset_name="${2}"
	local filename="${asset_name}"

	local url="https://github.com/alibaba/dragonwell11/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	else
		download_file "${url}" "${archive}" || return 1

		local regex
		if [[ "${filename}" = Alibaba_Dragonwell* ]];
		then
			# shellcheck disable=SC2016
			regex='s/^Alibaba_Dragonwell_([0-9\+].{1,}.*)_(?:(GA|Experimental|GA_Experimental|FP1)_)?(Linux|Windows)_(aarch64|x64)\.(.*)$/VERSION="$1" JAVA_VERSION="$1" RELEASE_TYPE="$2" OS="$3" ARCH="$4" EXT="$5"/g'
		else
			# shellcheck disable=SC2016
			regex='s/^OpenJDK(?:[0-9\+].{1,})_(x64|aarch64)_(linux|windows)_dragonwell_dragonwell-([0-9.]+)(?:_jdk)?[-_]([0-9._]+)-?(ga|.*)\.(tar\.gz|zip)$/ARCH="$1" OS="$2" VERSION="$3" JAVA_VERSION="$4" RELEASE_TYPE="$5" EXT="$6"/g'
		fi

		local VERSION=""
		local JAVA_VERSION=""
		local RELEASE_TYPE=""
		local OS=""
		local ARCH=""
		local EXT=""

		# Parse meta-data from file name
		eval "$(perl -pe "${regex}" <<< "${asset_name}")"

		if [[ -z "${RELEASE_TYPE}" ]]
		then
			RELEASE_TYPE='ga'
		fi

		if [[ "${VERSION}" =~ "preview" ]]
		then
			RELEASE_TYPE='ea'
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

RELEASE_FILE="${TEMP_DIR}/releases-${VENDOR}-11.json"
download_github_releases 'alibaba' 'dragonwell11' "${RELEASE_FILE}"

versions=$(jq -r '.[].tag_name' "${RELEASE_FILE}" | sort -V)
for version in ${versions}
do
	assets=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .assets[] | select(.content_type | startswith(\"application\")) | select(.name | contains(\"_source\") | not) | select(.name | endswith(\"jar\") | not) | .name" "${RELEASE_FILE}")
	for asset in ${assets}
	do
		download "${version}" "${asset}" || echo "Cannot download ${asset}"
	done
done

jq -s -S . "${METADATA_DIR}"/Alibaba_Dragonwell*.json > "${METADATA_DIR}/all.json"
