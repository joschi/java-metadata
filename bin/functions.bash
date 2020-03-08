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
	local cmd
	cmd="$(command -v "${hashalg}sum")"
	local checksum
	checksum=$("${cmd}" "${archive}" | cut -f 1 -d ' ')

	ensure_directory "${output_directory}"
	echo "${checksum}  ${filename}" > "${output_directory}/${filename}.${hashalg}"
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
	jo \
		vendor="${1}" \
		filename="${2}" \
		release_type="${3:-"ga"}" \
		version="${4}" \
		java_version="${5}" \
		jvm_impl="${6:-"hotspot"}" \
		os="${7}" \
		architecture="${8}" \
		file_type="${9}" \
		variant="${10}" \
		features="$(jo -a "${features[@]}" < /dev/null)" \
		url="${12}" \
		md5="${13}" \
		md5_file="${2}.md5" \
		sha1="${14}" \
		sha1_file="${2}.sha1" \
		sha256="${15}" \
		sha256_file="${2}.sha256" \
		sha512="${16}" \
		sha512_file="${2}.sha512" \
		size="${17}"
}

function normalize_os {
	case "${1}" in
	'linux') echo 'linux'
		;;
	'mac'|'macos'|'macosx'|'osx'|'darwin') echo 'macosx'
		;;
	'win'|'windows') echo 'windows'
		;;
	'solaris') echo 'solaris'
		;;
	'aix') echo 'aix'
		;;
	*) echo "unknown-os-${1}" ; return 1
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
	'arm'|'arm32') echo 'arm32'
		;;
	'arm32-vfp-hflt') echo 'arm32-vfp-hflt'
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
	*) echo "unknown-architecture-${1}" ; return 1
		;;
	esac
}

function find_supported_os {
	jq -r '.[].os' "$1" | sort | uniq
}

function find_supported_arch {
	jq -r '.[].architecture' "$1" | sort | uniq
}

function aggregate_metadata {
	local all_json="$1"
	local metadata_dir="$2"

	local supported_arch
	supported_arch=$(find_supported_arch "${all_json}")
	local supported_os
	supported_os=$(find_supported_os "${all_json}")
	local supported_variant
	supported_variant='jre jdk'

	for arch in $supported_arch
	do
		local arch_dir="${metadata_dir}/arch"
		ensure_directory "${arch_dir}"
		jq -S "[.[] | select(.architecture == \"${arch}\")]" "${all_json}" > "${arch_dir}/${arch}.json"

		local os_dir="${arch_dir}/${arch}/os"
		ensure_directory "${os_dir}"
		for os in $supported_os
		do
			jq -S "[.[] | select(.os == \"${os}\")]" "${arch_dir}/${arch}.json" > "${os_dir}/${os}.json"
		done
	done

	for os in $supported_os
	do
		local os_dir="${metadata_dir}/os"
		ensure_directory "${os_dir}"
		jq -S "[.[] | select(.os == \"${os}\")]" "${metadata_dir}/all.json" > "${os_dir}/${os}.json"

		local arch_dir="${os_dir}/${os}/arch"
		ensure_directory "${arch_dir}"
		for arch in $supported_arch
		do
			jq -S "[.[] | select(.architecture == \"${arch}\")]" "${os_dir}/${os}.json" > "${arch_dir}/${arch}.json"
		done
	done

	for variant in $supported_variant
	do
		local variant_dir="${metadata_dir}/${variant}"
		ensure_directory "${variant_dir}"
		jq -S "[.[] | select(.variant == \"${variant}\")]" "${all_json}" > "${variant_dir}/all.json"

		local arch_dir="${variant_dir}/arch"
		ensure_directory "${arch_dir}"
		for arch in $supported_arch
		do
			jq -S "[.[] | select(.architecture == \"${arch}\")]" "${variant_dir}/all.json" > "${arch_dir}/${arch}.json"

			local os_dir="${arch_dir}/${arch}/os"
			ensure_directory "${os_dir}"
			for os in $supported_os
			do
				jq -S "[.[] | select(.os == \"${os}\")]" "${arch_dir}/${arch}.json" > "${os_dir}/${os}.json"
			done
		done
	done
}
