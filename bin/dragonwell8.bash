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

	local url="https://github.com/dragonwell-project/dragonwell8/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	elif [[ "${filename}" =~ (tar\.gz|zip)$ ]]
	then
		download_file "${url}" "${archive}" || return 1

		local EDITION=""
		local VERSION=""
		local RELEASE_TYPE=""
		local OS=""
		local ARCH=""
		local EXT=""
		local FEATURES=""
		if [[ "${filename}" == 'Alibaba_Dragonwell8_Linux_x64_8.0-preview.tar.gz' ]]
		then
			VERSION="8.0.0-preview"
			RELEASE_TYPE="ea"
			OS="Linux"
			ARCH="x64"
			EXT="tar.gz"
		elif [[ "${filename}" =~ ^Alibaba_Dragonwell_8.6.6_ ]]
		then
			# shellcheck disable=SC2016
			local regex='s/^Alibaba_Dragonwell_([0-9].{1,})(_GA|Experimental|GA_Experimental|FP1)?_(x64|aarch64)_(Linux|linux|Windows|windows)\.(.*)$/VERSION="$1" RELEASE_TYPE="$2" ARCH="$3" OS="$4" EXT="$5"/g'

			# Parse meta-data from file name
			eval "$(perl -pe "${regex}" <<< "${asset_name}")"
		elif [[ "${filename}" =~ ^Alibaba_Dragonwell_(Standard|Extended) ]]
		then
			# shellcheck disable=SC2016
			local regex='s/^Alibaba_Dragonwell[-_](Standard|Extended)[-_]([0-9].{1,})[-_](x64|aarch64)[-_](Linux|linux|Windows|windows)\.(.*)$/EDITION="$1" VERSION="$2" ARCH="$3" OS="$4" EXT="$5"/g'

			# Parse meta-data from file name
			eval "$(perl -pe "${regex}" <<< "${asset_name}")"
		else
			# shellcheck disable=SC2016
			local regex='s/^Alibaba_Dragonwell_([0-9].{1,})[-_](GA|Experimental|GA_Experimental|FP1)_(Linux|linux|Windows|windows)_(x64|aarch64)\.(.*)$/VERSION="$1" RELEASE_TYPE="$2" OS="$3" ARCH="$4" EXT="$5"/g'

			# Parse meta-data from file name
			eval "$(perl -pe "${regex}" <<< "${asset_name}")"
		fi

		if [[ -z "${VERSION}" ]]
		then
			# shellcheck disable=SC2016
			regex='s/^Alibaba_Dragonwell_([0-9\+.]{1,}[^_]*)(?:_alpine)?_(aarch64|x64)_(Linux|linux|Windows|windows)\.(.*)$/VERSION="$1" JAVA_VERSION="$1" RELEASE_TYPE="jdk" OS="$3" ARCH="$2" EXT="$4"/g'
			eval "$(perl -pe "${regex}" <<< "${asset_name}")"
		fi

		if [[ -z "${VERSION}" ]]
		then
			echo "Unable to parse ${asset_name}"
			return 1
		fi

    if [[ "${EDITION}" = "Extended" ]]
    then
      FEATURES+="${FEATURES:+ }cloud"
    fi

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
	else
		echo "Skipping ${filename}"
	fi
}

RELEASE_FILE="${TEMP_DIR}/releases-${VENDOR}-8.json"
download_github_releases 'dragonwell-project' 'dragonwell8' "${RELEASE_FILE}"

versions=$(jq -r '.[].tag_name' "${RELEASE_FILE}" | sort -V)
for version in ${versions}
do
	assets=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .assets[] | select(.content_type | startswith(\"application\")) | select(.name | contains(\"_source\") | not) | select(.name | endswith(\"jar\") | not) | .name" "${RELEASE_FILE}")
	for asset in ${assets}
	do
		download "${version}" "${asset}" || echo "Cannot download ${asset}"
	done
done

jq -s -S . "${METADATA_DIR}"/{Alibaba_Dragonwell,OpenJDK}*.json > "${METADATA_DIR}/all.json"
