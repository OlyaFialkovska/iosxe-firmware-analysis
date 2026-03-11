#!/bin/bash
#
# October 2015, Chris Morin
#
# Copyright (c) 2015,2016 by Cisco Systems, Inc.
# All rights reserved
#

#
# This script finds the debug.conf file if a user specifies one. The user can
# specify a file on a tftp server, or a file on the local file system. To
# specify a file on a tftp server, the DEBUG_CONF variable needs to be set
# to tftp://<SERVER_IP>/<PATH_TO_FILE>
#
# This script behaves differently if run on the RP compared to the FP or CC.
# On the RP:
#     If the file is on a tftp server, it's copied to /tmp/debug.conf.
#     If the file is local, a symlink is created at /tmp/debug.conf which
#     points to it.
#     If no file exists, it makes an empty file at /tmp/debug.con
#
# On the FP and CC:
#     This script copies the debug.conf file from the active RP to
#     /tmp/debug.conf
#

source /common

set -eu -o pipefail

DEBUG_CONF_FILE="/tmp/debug.conf"

ACTIVE_RP="10.0.1.0"


function rp_find_debug_conf
{
    local tftp_re="^tftp://([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)(/.*)"

    local dest=$1

    # the debug.conf file location specified by the user is in
    # this file as an export statement ROMMON_DEBUG_CONF
    source /tmp/sw/boot/rmonbifo/env_var.sh
    if [[ -v ROMMON_DEBUG_CONF && -n $ROMMON_DEBUG_CONF ]]; then
        echo "User specified ${ROMMON_DEBUG_CONF} as debug.conf file"
        # see if it's a tftp address
        if [[ "$ROMMON_DEBUG_CONF" =~ $tftp_re ]]; then
            local server=${BASH_REMATCH[1]}
            local remote_path=${BASH_REMATCH[2]}
            tftp "$server" -c get "$remote_path" "$dest"

        # see if it's a local file
        elif [[ -f $ROMMON_DEBUG_CONF ]]; then
            ln -s "$ROMMON_DEBUG_CONF" "$dest"

        else
            echo "debug.conf file can't be found locally. Ignoring"
        fi
    fi

    # Touch the file unconditionally, creating an empty file if it doesn't
    # exist. This is necessary as the other FRUs assume a file is there.
    touch "$dest"
}

function find_debug_conf
{
    local fru=$1
    local dest=$2

    # For security, we remove existing attempts to set this file, in case of malicious
    # attempts to place the device into Developer Mode.
    rm -f $DEBUG_CONF_FILE

    if [[ "$fru" == "rp" ]]; then
        rp_find_debug_conf "$dest"
    else
        get_active_rp
        tftp $ACTIVE_RP tftp-private -c get "$dest" "$dest"
    fi

    chmod 444 "$dest"

    if [[ -s "$dest" ]]; then
        echo "--$dest------------------------"
        # show it, it is of value to us -- but nuke the comments, and
        # squeeze multiple blank lines into one.
        # the "|| true" is to not crash on files with no non-comment lines
        cat -s "$dest" | { grep -v "^#" || true; }
        echo "------------------------$dest--"
    fi
}

get_fru_env
find_debug_conf "$FRU" "$DEBUG_CONF_FILE"
