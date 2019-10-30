#!/usr/bin/env bash

set -e

BIN="openair-station"

BASE_URL="https://get.openair.city/station"

BIN_DIR="/usr/bin"

fatal() {
	echo ${*}
	exit 1
}

create_tmp_directory() {
	# Check if tmp is mounted as noexec
	if grep -Eq '^[^ ]+ /tmp [^ ]+ ([^ ]*,)?noexec[, ]' /proc/mounts > /dev/null 2>&1; then
		pattern="$(pwd)/${BIN}-install-XXXXXX"
	else
		pattern="/tmp/${BIN}-install-XXXXXX"
	fi

	mktemp -d $pattern
}

download() {
	url="${1}"
	dest="${2}"
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL --connect-timeout 10 --retry 3 "${url}" >"${dest}" || fatal "Cannot download ${url}"
	elif command -v wget >/dev/null 2>&1; then
		wget -T 15 -O - "${url}" >"${dest}" || fatal "Cannot download ${url}"
	else
		fatal "I need curl or wget to proceed, but neither is available on this system."
	fi
}

safe_sha256sum() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum $@
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 $@
	else
		fatal "I could not find a suitable checksum binary to use"
	fi
}

setup_arch() {
	SYSTEM="$(uname -s 2> /dev/null || uname -v)"
	OS="$(uname -o 2> /dev/null || uname -rs)"
	MACHINE="$(uname -m 2> /dev/null)"

	case ${MACHINE} in
	    x86_64)
		ARCH=amd64
		;;
	    arm*)
		ARCH=arm
		;;
	    *)
		fatal "Incompatible machine type: ${MACHINE}, exiting..."
		;;
	esac

	export BIN_ARCH="${BIN}.${ARCH}"
	export BIN_URL="${BASE_URL}/${BIN_ARCH}"
	export SHA_URL="${BASE_URL}/${BIN_ARCH}.sha256.txt"
}

umask 022

[[ "${UID}" -ne "0" ]] && fatal "$0 script must be run as root!"

export PATH="${PATH}:/usr/local/bin:/usr/local/sbin"

BIN_PATH="${BIN_DIR}/${BIN}"

TMPDIR=$(create_tmp_directory)
trap "rm -rf ${TMPDIR}" EXIT

cd ${TMPDIR} || :

setup_arch

download "${SHA_URL}" "${TMPDIR}/sha256sum.txt"

REMOTE_SHASUM=`awk '{ print $1 }' ${TMPDIR}/sha256sum.txt`
LOCAL_SHASUM=`safe_sha256sum ${BIN_PATH} | awk '{ print $1 }'`

if [[ "${REMOTE_SHASUM}" = "${LOCAL_SHASUM}" ]]; then
    exit 0
fi

download "${BIN_URL}" "${TMPDIR}/${BIN_ARCH}"

safe_sha256sum -c "${TMPDIR}/sha256sum.txt" >/dev/null 2>&1 || fatal "${BIN_ARCH} checksum validation failed."

mv "${TMPDIR}/${BIN_ARCH}" "${BIN_PATH}" || fatal "can't move ${BIN_ARCH} to ${BIN_PATH}"
chmod +x "${BIN_PATH}" || fatal "can't set executable bit to ${BIN_PATH}"

systemctl restart ${BIN}

exit 0
