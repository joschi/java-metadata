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

VENDOR='graalvm-community'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"

# Parse an asset file name and echo "JAVA_VERSION|OS|ARCH|EXT".
# Exits non-zero when the name does not match, so the caller can skip the asset
# instead of silently producing metadata with empty fields.
#
# Handles both naming schemes seen so far:
#   graalvm-community-jdk-17.0.7_linux-x64_bin.tar.gz        -> 17.0.7
#   graalvm-community-jdk-21.0.2_macos-aarch64_bin.tar.gz    -> 21.0.2
#   graalvm-community-jdk-25i3-25.0.4.1_windows-x64_bin.zip  -> 25.0.4.1
#
# The middle part between "jdk-" and "_<os>-<arch>" is treated as an opaque
# blob; the Java version is the trailing dotted-numeric run inside it. That way
# extra stream labels (25i3, and whatever comes next) do not break parsing, and
# version numbers with any number of components are accepted.
function parse_asset_name {
	perl -e '
		my $name = $ARGV[0];
		exit 1 unless $name =~ /^graalvm-community-jdk-(.+)_(linux|macos|windows)-(aarch64|x64)_bin\.(zip|tar\.gz)$/;
		my ($blob, $os, $arch, $ext) = ($1, $2, $3, $4);
		exit 1 unless $blob =~ /(\d+(?:\.\d+)*)$/;
		print join("|", $1, $os, $arch, $ext);
	' "${1}"
}

function download {
	local tag_name="${1}"
	local asset_name="${2}"
	local filename="${asset_name}"

	local url="https://github.com/graalvm/graalvm-ce-builds/releases/download/${tag_name}/${asset_name}"
	local metadata_file="${METADATA_DIR}/${filename}.json"
	local archive="${TEMP_DIR}/${filename}"

	if [[ -f "${metadata_file}" ]]
	then
		echo "Skipping ${filename}"
		return 0
	fi

	# Parse meta-data from file name
	local parsed
	if ! parsed="$(parse_asset_name "${asset_name}")"
	then
		echo "Cannot parse asset name ${asset_name}"
		return 1
	fi

	local JAVA_VERSION OS ARCH EXT
	IFS='|' read -r JAVA_VERSION OS ARCH EXT <<< "${parsed}"

	if [[ -z "${JAVA_VERSION}" || -z "${OS}" || -z "${ARCH}" || -z "${EXT}" ]]
	then
		echo "Incomplete meta-data parsed from ${asset_name}"
		return 1
	fi

	download_file "${url}" "${archive}" || return 1

	local json
	json="$(metadata_json \
		"${VENDOR}" \
		"${filename}" \
		'ga' \
		"${JAVA_VERSION}" \
		"${JAVA_VERSION}" \
		'graalvm' \
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
}

download_github_releases 'graalvm' 'graalvm-ce-builds' "${TEMP_DIR}/releases-graalvm-community.json"

# Release tags changed prefix: older releases use "jdk-21.0.2", newer ones use
# "graal-25.3.4.1". The real guard is the asset-name filter below, which only
# accepts assets starting with "graalvm-community" - so this grep only needs to
# be loose enough not to drop valid releases.
versions=$(jq -r '.[].tag_name' "${TEMP_DIR}/releases-graalvm-community.json" | sort -V | grep -E '^(jdk|graal)-' || true)

if [[ -z "${versions}" ]]
then
	echo "No matching release tags found - has the tag naming changed again?"
	exit 1
fi

for version in ${versions}
do
	assets=$(jq -r  ".[] | select(.tag_name == \"${version}\") | .assets[].name | select(startswith(\"graalvm-community\")) | select(endswith(\"tar.gz\") or endswith(\"zip\"))" "${TEMP_DIR}/releases-graalvm-community.json")
	for asset in ${assets}
	do
		download "${version}" "${asset}" || echo "Cannot download ${asset}"
	done
done

jq -s -S . "${METADATA_DIR}"/graalvm-community-*.json > "${METADATA_DIR}/all.json"
