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

function normalize_release_type {
	case "${1}" in
	"ca"|"ca-fx"|"") echo 'ga'
		;;
	"ea") echo 'ea'
		;;
	"ca-dbg"|"ca-fx-dbg"|"dbg") echo 'debug'
		;;
	*) return 1
		;;
	esac
}

function normalize_features {
	declare -a features
	if [[ "${1}" == "ca-fx" ]] || [[ "${1}" == "ca-fx-dbg" ]]
	then
		features+=("javafx")
	fi
	if [[ "${2}" == "musl_x64" ]]
	then
		features+=("musl")
	fi
	echo "${features[@]}"
}

# shellcheck disable=SC2016
REGEX='s/^jdk-([0-9+.]{2,})_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm)$/VERSION="$1" OS="$2" ARCH="$3" ARCHIVE="$4"/g'

function download_and_parse {
	MAJOR_VERSION="${1}"
	INDEX_FILE="${TEMP_DIR}/index${MAJOR_VERSION}.html"

	download_file "https://www.oracle.com/java/technologies/javase/jdk${MAJOR_VERSION}-archive-downloads.html" "${INDEX_FILE}"

	JDK_FILES=$(grep -o -E '<a href="https://download\.oracle\.com/java/.+/archive/(jdk-.+_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm))">' "${INDEX_FILE}" | perl -pe 's#<a href="https://download\.oracle\.com/java/.+/archive/(.+)">#$1#g' | sort -V)
	for JDK_FILE in ${JDK_FILES}
	do
		METADATA_FILE="${METADATA_DIR}/${JDK_FILE}.json"
		JDK_ARCHIVE="${TEMP_DIR}/${JDK_FILE}"
		JDK_URL="https://download.oracle.com/java/${MAJOR_VERSION}/archive/${JDK_FILE}"
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

for version in 17 18 19 20
do
	download_and_parse "$version"
done

jq -s -S . "${METADATA_DIR}"/jdk-*.json > "${METADATA_DIR}/all.json"
