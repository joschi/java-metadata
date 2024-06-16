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

VENDOR='oracle-graalvm'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

# shellcheck disable=SC2016
REGEX='s/^graalvm-jdk-([0-9+.]{2,})_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm)$/VERSION="$1" OS="$2" ARCH="$3" ARCHIVE="$4"/g'

function current_releases {
	local version="$1"

	# https://www.oracle.com/java/technologies/jdk-script-friendly-urls/
	local -a params=(
		'linux,aarch64,tar.gz'
		'linux,x64,tar.gz'
		'macos,aarch64,tar.gz'
		'macos,x64,tar.gz'
		'windows,x64,zip'
		)
	for param in "${params[@]}"
	do
		local os
		os=$(cut -f1 -d, <<<"$param")
		local arch
		arch=$(cut -f2 -d, <<<"$param")
		local ext
		ext=$(cut -f3 -d, <<<"$param")

		echo "graalvm-jdk-${version}_${os}-${arch}_bin.${ext}"
	done
}

function download_and_parse {
	MAJOR_VERSION="${1}"
	INDEX_FILE="${TEMP_DIR}/index${MAJOR_VERSION}.html"

	download_file "https://www.oracle.com/java/technologies/javase/graalvm-jdk${MAJOR_VERSION}-archive-downloads.html" "${INDEX_FILE}"

	JDK_FILES=$(grep -o -E '<a href="https://download\.oracle\.com/graalvm/.+/archive/(graalvm-jdk-.+_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm))">' "${INDEX_FILE}" | perl -pe 's#<a href="https://download\.oracle\.com/graalvm/.+/archive/(.+)">#$1#g' | sort -V)

	CURRENT_RELEASE=$(curl -sSf https://www.oracle.com/java/technologies/downloads/ | (grep "<h3 id=\"graalvmjava${MAJOR_VERSION}\"" || true) | perl -pe 's#<h3 id="graalvmjava[0-9]{2}">GraalVM for JDK (.+) downloads</h3>#$1#g')
	JDK_FILES_CURRENT=$(current_releases "${CURRENT_RELEASE}")

	for JDK_FILE in ${JDK_FILES} ${JDK_FILES_CURRENT}
	do
		METADATA_FILE="${METADATA_DIR}/${JDK_FILE}.json"
		JDK_ARCHIVE="${TEMP_DIR}/${JDK_FILE}"
		JDK_URL="https://download.oracle.com/graalvm/${MAJOR_VERSION}/archive/${JDK_FILE}"
		if [[ -f "${METADATA_FILE}" ]]
		then
			echo "Skipping ${JDK_FILE}"
		else
			download_file "${JDK_URL}" "${JDK_ARCHIVE}"
			VERSION=""
			OS=""
			ARCH=""
			ARCHIVE=""

			# Parse meta-data from file name
			PARSED_NAME=$(perl -pe "${REGEX}" <<< "${JDK_FILE}")
			if [[ "${PARSED_NAME}" = "${JDK_FILE}" ]]
			then
				echo "Regular expression didn't match ${JDK_FILE}"
				continue
			else
				eval "${PARSED_NAME}"
			fi

			METADATA_JSON="$(metadata_json \
				"${VENDOR}" \
				"${JDK_FILE}" \
				"ga" \
				"${VERSION}" \
				"${VERSION}" \
				'graalvm' \
				"$(normalize_os "${OS}")" \
				"$(normalize_arch "${ARCH}")" \
				"${ARCHIVE}" \
				"jdk" \
				"" \
				"${JDK_URL}" \
				"$(hash_file 'md5' "${JDK_ARCHIVE}" "${CHECKSUM_DIR}")" \
				"$(hash_file 'sha1' "${JDK_ARCHIVE}" "${CHECKSUM_DIR}")" \
				"$(hash_file 'sha256' "${JDK_ARCHIVE}" "${CHECKSUM_DIR}")" \
				"$(hash_file 'sha512' "${JDK_ARCHIVE}" "${CHECKSUM_DIR}")" \
				"$(file_size "${JDK_ARCHIVE}")" \
				"${JDK_FILE}"
			)"

			echo "${METADATA_JSON}" > "${METADATA_FILE}"
			rm -f "${JDK_ARCHIVE}"
		fi
	done
}

for version in 17 20 21 22
do
	download_and_parse "$version"
done

jq -s -S . "${METADATA_DIR}"/graalvm-*.json > "${METADATA_DIR}/all.json"
