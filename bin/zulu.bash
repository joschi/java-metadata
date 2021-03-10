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

VENDOR='zulu'
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
REGEX='s/^zulu([0-9+_.]{2,})-(?:(ca-fx-dbg|ca-fx|ca-hl|ca-dbg|ea-cp3|ca|ea|dbg|oem)-)?(jdk|jre)(.*)-(linux|macosx|win|solaris)_(musl_x64|x64|i686|aarch32hf|aarch32sf|aarch64|ppc64|sparcv9)\.(.*)$/VERSION="$1" RELEASE_TYPE="$2" IMAGE_TYPE="$3" JAVA_VERSION="$4" OS="$5" ARCH="$6" ARCHIVE="$7"/g'

INDEX_FILE="${TEMP_DIR}/index.html"
download_file 'https://static.azul.com/zulu/bin/' "${INDEX_FILE}"

ZULU_FILES=$(grep -o -E '<a href="(zulu.+-(linux|macosx|win|solaris)_(musl_x64|x64|i686|aarch32hf|aarch32sf|aarch64|ppc64|sparcv9)\.(tar\.gz|zip|msi|dmg))">' "${INDEX_FILE}" | perl -pe 's/<a href="(.+)">/$1/g' | sort -V)
for ZULU_FILE in ${ZULU_FILES}
do
	METADATA_FILE="${METADATA_DIR}/${ZULU_FILE}.json"
	ZULU_ARCHIVE="${TEMP_DIR}/${ZULU_FILE}"
	ZULU_URL="https://static.azul.com/zulu/bin/${ZULU_FILE}"
	if [[ -f "${METADATA_FILE}" ]]
	then
		echo "Skipping ${ZULU_FILE}"
	else
		download_file "${ZULU_URL}" "${ZULU_ARCHIVE}"
		RELEASE_TYPE=""
		VERSION=""
		JAVA_VERSION=""
		IMAGE_TYPE=""
		OS=""
		ARCH=""
		ARCHIVE=""

		# Parse meta-data from file name
		PARSED_NAME=$(perl -pe "${REGEX}" <<< "${ZULU_FILE}")
		if [[ "${PARSED_NAME}" = "${ZULU_FILE}" ]]
		then
			echo "Regular expression didn't match ${ZULU_FILE}"
			continue
		else
			eval "${PARSED_NAME}"
		fi

		FEATURES="$(normalize_features "${RELEASE_TYPE}" "${ARCH}")"
		if [[ "${ARCH}" = 'musl_x64' ]]
		then
			ARCH='x64'
		fi

		METADATA_JSON="$(metadata_json \
			"${VENDOR}" \
			"${ZULU_FILE}" \
			"$(normalize_release_type "${RELEASE_TYPE}")" \
			"${VERSION}" \
			"${JAVA_VERSION}" \
			'hotspot' \
			"$(normalize_os "${OS}")" \
			"$(normalize_arch "${ARCH}")" \
			"${ARCHIVE}" \
			"${IMAGE_TYPE}" \
			"${FEATURES}" \
			"${ZULU_URL}" \
			"$(hash_file 'md5' "${ZULU_ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha1' "${ZULU_ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha256' "${ZULU_ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(hash_file 'sha512' "${ZULU_ARCHIVE}" "${CHECKSUM_DIR}")" \
			"$(file_size "${ZULU_ARCHIVE}")" \
			"${ZULU_FILE}"
		)"

		echo "${METADATA_JSON}" > "${METADATA_FILE}"
		rm -f "${ZULU_ARCHIVE}"
	fi
done

jq -s -S . "${METADATA_DIR}"/zulu*.json > "${METADATA_DIR}/all.json"
