#!/usr/bin/env bash

# Based on https://my-netdata.io/kickstart.sh

set -e

BIN="openair-station"

BASE_URL="https://get.openair.city/station"

BIN_DIR="/usr/bin"

setup_terminal() {
	TPUT_RESET=""
	TPUT_YELLOW=""
	TPUT_WHITE=""
	TPUT_BGRED=""
	TPUT_BGGREEN=""
	TPUT_BOLD=""
	TPUT_DIM=""

	# Is stderr on the terminal? If not, then fail
	test -t 2 || return 1

	if command -v tput >/dev/null 2>&1; then
		if [[ $(($(tput colors 2>/dev/null))) -ge 8 ]]; then
			# Enable colors
			TPUT_RESET="$(tput sgr 0)"
			TPUT_YELLOW="$(tput setaf 3)"
			TPUT_WHITE="$(tput setaf 7)"
			TPUT_BGRED="$(tput setab 1)"
			TPUT_BGGREEN="$(tput setab 2)"
			TPUT_BOLD="$(tput bold)"
			TPUT_DIM="$(tput dim)"
		fi
	fi

	return 0
}
setup_terminal || echo >/dev/null

run_ok() {
	printf >&2 "${TPUT_BGGREEN}${TPUT_WHITE}${TPUT_BOLD} OK ${TPUT_RESET} ${*} \n\n"
}

run_failed() {
	printf >&2 "${TPUT_BGRED}${TPUT_WHITE}${TPUT_BOLD} FAILED ${TPUT_RESET} ${*} \n\n"
}

ESCAPED_PRINT_METHOD=
printf "%q " test >/dev/null 2>&1
[[ $? -eq 0 ]] && ESCAPED_PRINT_METHOD="printfq"
escaped_print() {
	if [[ "${ESCAPED_PRINT_METHOD}" = "printfq" ]]; then
		printf "%q " "${@}"
	else
		printf "%s" "${*}"
	fi
	return 0
}

run_logfile="/dev/null"
run() {
	local user="${USER--}" dir="${PWD}" info info_console

	if [[ "${UID}" = "0" ]]; then
		info="[root ${dir}]# "
		info_console="[${TPUT_DIM}${dir}${TPUT_RESET}]# "
	else
		info="[${user} ${dir}]$ "
		info_console="[${TPUT_DIM}${dir}${TPUT_RESET}]$ "
	fi

	printf >>"${run_logfile}" "${info}"
	escaped_print >>"${run_logfile}" "${@}"
	printf >>"${run_logfile}" " ... "

	printf >&2 "${info_console}${TPUT_BOLD}${TPUT_YELLOW}"
	escaped_print >&2 "${@}"
	printf >&2 "${TPUT_RESET}\n"

	"${@}"

	local ret=$?
	if [[ ${ret} -ne 0 ]]; then
		run_failed
		printf >>"${run_logfile}" "FAILED with exit code ${ret}\n"
	else
		run_ok
		printf >>"${run_logfile}" "OK\n"
	fi

	return ${ret}
}

progress() {
	echo >&2 " --- ${TPUT_DIM}${TPUT_BOLD}${*}${TPUT_RESET} --- "
}

fatal() {
	printf >&2 "${TPUT_BGRED}${TPUT_WHITE}${TPUT_BOLD} ABORTED ${TPUT_RESET} ${*} \n\n"
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
		run curl -fsSL --connect-timeout 10 --retry 3 "${url}" >"${dest}" || fatal "Cannot download ${url}"
	elif command -v wget >/dev/null 2>&1; then
		run wget -T 15 -O - "${url}" >"${dest}" || fatal "Cannot download ${url}"
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
	    aarch64)
		ARCH=arm64
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

sudo=""
[[ -z "${UID}" ]] && UID="$(id -u)"
[[ "${UID}" -ne "0" ]] && sudo="sudo"

export PATH="${PATH}:/usr/local/bin:/usr/local/sbin"

TMPDIR=$(create_tmp_directory)
trap "rm -rf ${TMPDIR}" EXIT

cd ${TMPDIR} || :

UPD_URL="${BASE_URL}/update.sh"
UPD_BIN="${BIN}-update"

progress "Installing ${BIN}..."

setup_arch

download "${BIN_URL}" "${TMPDIR}/${BIN_ARCH}"
download "${SHA_URL}" "${TMPDIR}/sha256sum.txt"
download "${UPD_URL}" "${TMPDIR}/${UPD_BIN}"

safe_sha256sum -c "${TMPDIR}/sha256sum.txt" >/dev/null 2>&1 || fatal "${BIN_ARCH} checksum validation failed."

BIN_PATH="${BIN_DIR}/${BIN}"

run ${sudo} mv "${TMPDIR}/${BIN_ARCH}" "${BIN_PATH}" || fatal "can't move ${BIN_ARCH} to ${BIN_PATH}"
run ${sudo} chmod +x "${BIN_PATH}" || fatal "can't set executable bit to ${BIN_PATH}"

progress "Installing ${UPD_BIN}..."

run ${sudo} mv "${TMPDIR}/${UPD_BIN}" "${BIN_DIR}" || fatal "can't move ${UPD_BIN} to ${BIN_DIR}"
run ${sudo} chmod +x "${BIN_DIR}/${UPD_BIN}" || fatal "can't set executable bit to ${UPD_BIN}"

progress "Setting up systemd..."
run ${sudo} sh -c "cat > /etc/systemd/system/${BIN}.service" <<EOF
[Unit]
Description=OpenAir Station
After=network.target

[Service]
ExecStart=${BIN_PATH} \$STATION_EXEC_OPTIONS
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

progress "Setting up cron..."
MINUTES=$(( RANDOM % 59 ))
run ${sudo} sh -c "cat > /etc/cron.d/${UPD_BIN}" <<EOF
# check daily after midnight
${MINUTES} 0 * * * root ${BIN_DIR}/${UPD_BIN}
EOF

run ${sudo} systemctl daemon-reload
run ${sudo} systemctl enable ${BIN}

progress "Starting up ${BIN}..."
run ${sudo} systemctl restart ${BIN}

progress "Done!"
