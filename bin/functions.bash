#!/usr/bin/env bash
set -e
set -Euo pipefail

function ensure_directory {
	local dir="${1}"
	if [ ! -d "${dir}" ]
	then
		mkdir -p "${dir}"
	fi
}

function download_github_releases {
	local org="${1}"
	local repo="${2}"
	local output_file="${3}"
	local args=('--silent' '--show-error' '--fail' '-H' 'Accept: application/json'  '--output' "${output_file}")
	if [[ -n "${GITHUB_API_TOKEN:-}" ]]; then
		args+=('-H' "Authorization: token ${GITHUB_API_TOKEN}")
	fi
	curl "${args[@]}" "https://api.github.com/repos/${org}/${repo}/releases"
}

function download_file {
	local url="${1}"
	local output_file="${2}"
	local args=('--silent' '--show-error' '--fail' '--location' '--output' "${output_file}")
	echo "Downloading ${url}"
	curl "${args[@]}" "${url}"
}

function check_url_exists {
	local url="${1}"
	local args=('--silent' '--show-error' '--fail' '--location' '--head')
	curl "${args[@]}" "${url}"
}

function hash_file {
	local hashalg="${1}"
	local archive="${2}"
	local output_directory="${3}"
	local filename
	filename="$(basename "${archive}")"
	local real_filename
	if [[ $# == 4 && -n "${4}" ]]
	then
		real_filename="${4}"
	else
		real_filename="${filename}"
	fi
	local cmd
	cmd="$(command -v "${hashalg}sum")"
	local checksum
	checksum=$("${cmd}" "${archive}" | cut -f 1 -d ' ')

	ensure_directory "${output_directory}"
	echo "${checksum}  ${real_filename}" > "${output_directory}/${filename}.${hashalg}"
	echo "${checksum}"
}

function metadata_file {
	local vendor="${1}"
	local filename="${2}"
	local directory="${3}"
	echo "${directory}/${vendor}/${filename}"
}

function metadata_exists {
	local vendor="${1}"
	local filename="${2}"
	local directory="${2}"
	local path
	path=$(metadata_file "${vendor}" "${filename}" "${directory}")

	if [[ -f "${path}" ]]
	then
		return 0
	else
		return 1
	fi
}

function file_size {
	stat -c '%s' "$1"
}

function metadata_json {
	declare -a features
	for item in ${11}
	do
		features+=("${item}")
	done
	jo -- \
		vendor="${1}" \
		filename="${2}" \
		release_type="${3:-"ga"}" \
		-s version="${4}" \
		-s java_version="${5}" \
		jvm_impl="${6:-"hotspot"}" \
		os="${7}" \
		architecture="${8}" \
		file_type="${9}" \
		image_type="${10}" \
		features="$(jo -a "${features[@]}" < /dev/null)" \
		url="${12}" \
		-s md5="${13}" \
		md5_file="${18}.md5" \
		-s sha1="${14}" \
		sha1_file="${18}.sha1" \
		-s sha256="${15}" \
		sha256_file="${18}.sha256" \
		-s sha512="${16}" \
		sha512_file="${18}.sha512" \
		size="${17}"
}

function normalize_os {
	case "${1}" in
	'linux'|'Linux'|'alpine-linux') echo 'linux'
		;;
	'mac'|'macos'|'macosx'|'osx'|'darwin'|'macOS') echo 'macosx'
		;;
	'win'|'windows'|'Windows') echo 'windows'
		;;
	'solaris') echo 'solaris'
		;;
	'aix') echo 'aix'
		;;
	*) echo "unknown-os-${1}" ; exit 1
		;;
	esac
}

function normalize_arch {
	case "${1}" in
	'amd64'|'x64'|'x86_64') echo 'x86_64'
		;;
	'x32'|'x86'|'i386'|'i586'|'i686') echo 'i686'
		;;
	'aarch64'|'arm64') echo 'aarch64'
		;;
	'arm'|'arm32'|'armv7'|'aarch32sf') echo 'arm32'
		;;
	'arm32-vfp-hflt'|'aarch32hf') echo 'arm32-vfp-hflt'
		;;
	'ppc64') echo 'ppc64'
		;;
	'ppc64le') echo 'ppc64le'
		;;
	's390') echo 's390'
		;;
	's390x') echo 's390x'
		;;
	'sparcv9') echo 'sparcv9'
		;;
	*) echo "unknown-architecture-${1}" ; exit 1
		;;
	esac
}

function find_supported_os {
	jq -r '.[].os' "$1" | sort | uniq
}

function find_supported_arch {
	jq -r '.[].architecture' "$1" | sort | uniq
}

function find_supported_vendors {
	jq -r '.[].vendor' "$1" | sort | uniq
}

function aggregate_metadata {
	local all_json="$1"
	local metadata_dir="$2"

	local supported_arch
	supported_arch=$(find_supported_arch "${all_json}")
	local supported_os
	supported_os=$(find_supported_os "${all_json}")
	local supported_image_type='jre jdk'
	local release_types='ea ga'
	local jvm_impls='hotspot openj9 graalvm'
	local vendors
	vendors=$(find_supported_vendors "${all_json}")

	# https://api.adoptopenjdk.net/swagger-ui/
	# /v3/binary/latest/{feature_version}/{release_type}/{os}/{arch}/{image_type}/{jvm_impl}/{heap_size}/{vendor}
	for release_type in $release_types
	do
		local release_type_dir="${metadata_dir}/${release_type}"
		ensure_directory "${release_type_dir}"
		jq -S "[.[] | select(.release_type == \"${release_type}\")]" "${all_json}" > "${release_type_dir}/../${release_type}.json"

		for os in $supported_os
		do
			local os_dir="${release_type_dir}/${os}"
			ensure_directory "${os_dir}"
			jq -S "[.[] | select(.os == \"${os}\")]" "${release_type_dir}/../${release_type}.json" > "${os_dir}/../${os}.json"

			for arch in $supported_arch
			do
				local arch_dir="${os_dir}/${arch}"
				ensure_directory "${arch_dir}"
				jq -S "[.[] | select(.architecture == \"${arch}\")]" "${os_dir}/../${os}.json" > "${arch_dir}/../${arch}.json"

				for image_type in $supported_image_type
				do
					local image_type_dir="${arch_dir}/${image_type}"
					ensure_directory "${image_type_dir}"
					jq -S "[.[] | select(.image_type == \"${image_type}\")]" "${arch_dir}/../${arch}.json" > "${image_type_dir}/../${image_type}.json"

					for jvm_impl in $jvm_impls
					do
						local jvm_impl_dir="${image_type_dir}/${jvm_impl}"
						ensure_directory "${jvm_impl_dir}"
						jq -S "[.[] | select(.jvm_impl == \"${jvm_impl}\")]" "${image_type_dir}/../${image_type}.json" > "${jvm_impl_dir}/../${jvm_impl}.json"

						for vendor in $vendors
						do
							local vendor_dir="${jvm_impl_dir}/${vendor}"
							ensure_directory "${vendor_dir}"
							jq -S "[.[] | select(.vendor == \"${vendor}\")]" "${jvm_impl_dir}/../${jvm_impl}.json" > "${vendor_dir}/../${vendor}.json"
						done
					done
				done
			done
		done

	done
}
