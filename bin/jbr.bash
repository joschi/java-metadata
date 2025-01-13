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

VENDOR='jetbrains'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"


function download {
	local asset_name="${1}"
	local url="${2}"
	local release_type="${3}"
	local description="${4}"
	local filename="${asset_name}"

	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	echo "asset ${asset_name} : |${description}|"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
	elif [[ "${filename}" =~ (tar\.gz|zip|pkg)$ ]]
	then
		download_file "${url}" "${archive}" || return 1

		# shellcheck disable=SC2016
		local regex='s/^jbr(sdk)?(?:_\w+)?-([0-9\+.]{1,})-(linux-musl|linux|osx|macos|windows)-(aarch64|x64|x86)(?:-\w+)?-(b[0-9\+.]{1,})(?:_\w+)?\.(tar\.gz|zip|pkg)$/ARCH="$4" OS="$3" VERSION="$2$5" JAVA_VERSION="$3" IMAGE_TYPE="$1" EXT="$6"/g'

		local VERSION=""
		local JAVA_VERSION=""
		local RELEASE_TYPE="$release_type"
		local OS=""
		local ARCH=""
		local EXT=""
		local FEATURES=""
		local IMAGE_TYPE=""

		# Parse meta-data from file name
		eval "$(perl -pe "${regex}" <<< "${asset_name}")"


		if [[ -z "${IMAGE_TYPE}" ]]
		then
			IMAGE_TYPE='jre'
		else
			IMAGE_TYPE='jdk'
		fi

		if [[ "${description}" =~ "fastdebug" ]]
		then
			FEATURES="$FEATURES fastdebug"
		fi

		if [[ "${description}" =~ "debug symbols" ]]
		then
			FEATURES="$FEATURES debug"
		fi

		if [[ "${description}" =~ "FreeType" ]]
		then
			FEATURES="$FEATURES freetype"
		fi

		if [[ "${description}" =~ "JCEF" ]]
		then
			FEATURES="$FEATURES jcef"
		fi

		if [[ "${OS}" = "linux-musl" ]]
		then
			FEATURES="$FEATURES musl"
			OS='linux'
		fi

		local json
		json="$(metadata_json \
			"${VENDOR}" \
			"${filename}" \
			"${RELEASE_TYPE}" \
			"${VERSION}" \
			"${JAVA_VERSION}" \
			'hotspot' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${EXT}" \
			"${IMAGE_TYPE}" \
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

RELEASE_FILE="${TEMP_DIR}/releases-${VENDOR}.json"
download_github_releases 'JetBrains' 'JetBrainsRuntime' "${RELEASE_FILE}"

versions=$(jq -r '.[].tag_name' "${RELEASE_FILE}" | sort -V)

assets_json=${TEMP_DIR}/assets-parsed.json
jq  '[ .[] | ({version: .tag_name, type: (if .prerelease then "ea" else "ga" end)  }  + (.body | capture("\\|\\s*(?:\\*\\*)?(?<description>[^|]+?)(?:\\*\\*)?\\s*\\|\\s*\\[(?<file>[^\\]]+)\\]\\((?<url>https:[^\\)]+)\\)\\s*\\|\\s*\\[checksum\\]\\((?<checksum_url>https:[^\\)]+)\\)";"g")) ) ]' "$RELEASE_FILE" > "${assets_json}"

for version in ${versions}
do

	readarray -t assets < <(jq -r  ".[] | select(.version == \"${version}\") | .file" "${assets_json}")
	readarray -t download_urls < <(jq -r  ".[] | select(.version == \"${version}\") | .url" "${assets_json}")
	readarray -t release_types < <(jq -r  ".[] | select(.version == \"${version}\") | .type" "${assets_json}")
	readarray -t descriptions < <(jq -r ".[] | select(.version == \"${version}\") | .description" "${assets_json}")
	for ((i = 0; i < ${#assets[@]}; i++));
	do
		download "${assets[i]}" "${download_urls[i]}" "${release_types[i]}" "${descriptions[i]}" || echo "Cannot download ${asset_name}"
	done

done

jq -s -S . "${METADATA_DIR}"/jbr*.json > "${METADATA_DIR}/all.json"
