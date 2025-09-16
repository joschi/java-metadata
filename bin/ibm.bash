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

VENDOR='ibm'
METADATA_DIR="${1}/${VENDOR}"
CHECKSUM_DIR="${2}/${VENDOR}"

ensure_directory "${METADATA_DIR}"
ensure_directory "${CHECKSUM_DIR}"


INDEX_FILE="${TEMP_DIR}/index.html"
download_file 'https://public.dhe.ibm.com/ibmdl/export/pub/systems/cloud/runtimes/java/' "${INDEX_FILE}"

JDK_VERSIONS=$(grep -o -E '<a href="([8]\.[01]\.[0-9]+\.[0-9]+)/">' "${INDEX_FILE}" | perl -pe 's#<a href="([78][^/]+)/">#$1#g' | sort -V)

for JDK_VERSION in ${JDK_VERSIONS}
do
  download_file "https://public.dhe.ibm.com/ibmdl/export/pub/systems/cloud/runtimes/java/${JDK_VERSION}/linux/" "${INDEX_FILE}"
  ARCHITECTURES=$(grep -o -E '<a href="([a-z0-9_]+)/">' "${INDEX_FILE}" | perl -pe 's#<a href="([a-z0-9_]+)/">#$1#g' | sort -V)
  for ARCHITECTURE in ${ARCHITECTURES}
  do
    download_file "https://public.dhe.ibm.com/ibmdl/export/pub/systems/cloud/runtimes/java/${JDK_VERSION}/linux/${ARCHITECTURE}/" "${INDEX_FILE}"
    IBM_FILES=$(grep -o -E '<a href="(.*\.tgz)">' "${INDEX_FILE}" | perl -pe 's#<a href="(.*\.tgz)">#$1#g' | sort -V)
    for IBM_FILE in ${IBM_FILES}
    do
      if [[ "${IBM_FILE}" == *"sfj"* ]]
      then
        echo  "Ignoring ${IBM_FILE}"
      else
        METADATA_FILE="${METADATA_DIR}/${IBM_FILE}.json"
        IBM_ARCHIVE="${TEMP_DIR}/${IBM_FILE}"

        RELEASE_TYPE="ga"
        VERSION=$JDK_VERSION
        JAVA_VERSION=$JDK_VERSION
        JVM_IMPL="openj9"
        OS="linux"
        ARCH=$ARCHITECTURE
        ARCHIVE=$(echo "$IBM_FILE" | perl -pe 's#.*\.([^.]+)$#$1#g')
        if [[ "${IBM_FILE}" = *"sdk"* ]]
        then
          IMAGE_TYPE="jdk"
        else
          IMAGE_TYPE="jre"
        fi
        FEATURES=""
        IBM_URL="https://public.dhe.ibm.com/ibmdl/export/pub/systems/cloud/runtimes/java/${JDK_VERSION}/linux/${ARCHITECTURE}/${IBM_FILE}"

        download_file "${IBM_URL}" "${IBM_ARCHIVE}"

        METADATA_JSON="$(metadata_json \
          "${VENDOR}" \
          "${IBM_FILE}" \
          "${RELEASE_TYPE}" \
          "${VERSION}" \
          "${JAVA_VERSION}" \
          "${JVM_IMPL}" \
          "${OS}" \
          "$(normalize_arch "${ARCH}")" \
          "${ARCHIVE}" \
          "${IMAGE_TYPE}" \
          "${FEATURES}" \
          "${IBM_URL}" \
          "$(hash_file 'md5' "${IBM_ARCHIVE}" "${CHECKSUM_DIR}")" \
          "$(hash_file 'sha1' "${IBM_ARCHIVE}" "${CHECKSUM_DIR}")" \
          "$(hash_file 'sha256' "${IBM_ARCHIVE}" "${CHECKSUM_DIR}")" \
          "$(hash_file 'sha512' "${IBM_ARCHIVE}" "${CHECKSUM_DIR}")" \
          "$(file_size "${IBM_ARCHIVE}")" \
          "${IBM_FILE}"
        )"

        echo "${METADATA_JSON}" > "${METADATA_FILE}"
        rm -f "${IBM_ARCHIVE}"

      fi
    done

  done


done

jq -s -S . "${METADATA_DIR}"/ibm-*.json > "${METADATA_DIR}/all.json"
