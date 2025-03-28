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

VENDOR='oracle'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

# shellcheck disable=SC2016
REGEX='s/^jdk-([0-9+.]{2,})_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm)$/VERSION="$1" OS="$2" ARCH="$3" ARCHIVE="$4"/g'

function download_and_parse {
	MAJOR_VERSION="${1}"
	INDEX_FILE="${TEMP_DIR}/index${MAJOR_VERSION}.html"

	download_file "https://www.oracle.com/java/technologies/javase/jdk${MAJOR_VERSION}-archive-downloads.html" "${INDEX_FILE}"

	JDK_FILES=$(grep -o -E '<a href="https://download\.oracle\.com/java/.+/archive/(jdk-.+_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm))">' "${INDEX_FILE}" | perl -pe 's#<a href="https://download\.oracle\.com/java/.+/archive/(.+)">#$1#g' | sort -V)

	for JDK_FILE in ${JDK_FILES}
	do
		if [[ -z "${JDK_FILE}" ]]
		then
			continue
		fi

		METADATA_FILE="${METADATA_DIR}/${JDK_FILE}.json"
		JDK_ARCHIVE="${TEMP_DIR}/${JDK_FILE}"
		JDK_URL="https://download.oracle.com/java/${MAJOR_VERSION}/archive/${JDK_FILE}"
		if [[ -f "${METADATA_FILE}" ]]
		then
			echo "Skipping ${JDK_FILE}"
		else
			if ! download_file "${JDK_URL}" "${JDK_ARCHIVE}";
			then
				echo "Failed to download ${JDK_FILE}, skipping"
				continue
			fi
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
				'hotspot' \
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

# Latest
download_file "https://java.oraclecloud.com/javaVersions" "${TEMP_DIR}/latest-versions.json"
for version in $(jq -r '.items[].latestReleaseVersion' "${TEMP_DIR}/latest-versions.json")
do
	download_file "https://java.oraclecloud.com/javaReleases/${version}" "${TEMP_DIR}/release-${version}.json"
	LICENSE_TYPE=$(jq -r '.licenseDetails.licenseType' "${TEMP_DIR}/release-${version}.json")
	if [[ "$LICENSE_TYPE" = "OTN" ]]
	then
		echo "Skipping OTN licensed versions"
		continue
	fi

	for JDK_URL in $(jq -r '.artifacts[].downloadUrl' "${TEMP_DIR}/release-${version}.json")
	do
		JDK_FILE=$(basename "${JDK_URL}")
		JDK_ARCHIVE="${TEMP_DIR}/${JDK_FILE}"
		METADATA_FILE="${METADATA_DIR}/${JDK_FILE}.json"
		if [[ -f "${METADATA_FILE}" ]]
		then
			echo "Skipping ${JDK_FILE}"
		else
			if ! download_file "${JDK_URL}" "${JDK_ARCHIVE}";
			then
				echo "Failed to download ${JDK_FILE}, skipping"
				continue
			fi
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
				'hotspot' \
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
done

for version in 17 18 19 20 21 22 23
do
	download_and_parse "$version"
done

jq -s -S . "${METADATA_DIR}"/jdk-*.json > "${METADATA_DIR}/all.json"
