#!/bin/bash
#------------------------------------------------------------------
# Copyright (c) 2019 by cisco Systems, Inc.
# All rights reserved.
#------------------------------------------------------------------
#
# This script's purpose is to allow multiple platforms share the same image,
# while still being able to have platform specific configuration/functionality.
# This script specifically deals with differences that need to be represented
# as file differences.
#
# As an example, consider two platforms with different persistent storage
# partition schemes. The persistent storage mounts are defined in /etc/fstab,
# Since there can be only one /etc/fstab per image, we run into the issue of
# how to we each platform to have it's own partitioning scheme while sharing
# an image.
#
#
# We solve this issue by defining the mapping from platform to files on the
# image. This script determines the platform and creates symlinks in the root
# filesystem pointing to the platform's files.
#
# This script is flexible in how specific "platform" is. It can refer to the
# specific fru, or the chassis as a whole. The script uses the most specific
# mapping that exists for the given platform.
#

function graft_to_root
{
    cp -srf "${1}"/* /
}

BOARD_TYPE=$( cat /sys/bus/platform/devices/cpld/board_type )
BOARD_SUBTYPE=$( cat /sys/bus/platform/devices/cpld/board_subtype )

board_type=${BOARD_TYPE,,}
board_subtype=${BOARD_SUBTYPE,,}


echo "board type: $board_type"
echo "board subtype: $board_subtype"

if [[ $board_subtype == "imperial" ]]; then
    graft_to_root "/platform-specific/${board_subtype}"
    echo "Grafting for passport for now (FAKE_IMPERIAL)"
    board_subtype="passport"
fi

fru_dir="/platform-specific/${board_type}-${board_subtype}"
chassis_dir="/platform-specific/${board_subtype}"

if [[ -d $fru_dir ]]; then
    echo "Grafting $fru_dir"
    graft_to_root "$fru_dir"
elif [[ -d "$chassis_dir" ]]; then
    echo "Grafting $chassis_dir"
    graft_to_root "$chassis_dir"
else
    echo "Didn't find a platform specific directory to graft"
fi

find /etc/systemd/system -maxdepth 1 -type l -exec rm {} \;
systemctl preset-all
systemctl --no-reload set-default binos.target
