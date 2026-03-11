#!/bin/bash
#
# Copyright (c) 2016-2018 by Cisco Systems, Inc.
# All rights reserved
#
# This script verifies that the partition table on persistent storage is
# correct and repartitions it if it isn't.

# To have this script run for a given device, you need to add a symlink from
# /lib/systemd/system/local-fs.target.requires/check_partitions@<device>.service
# pointing to
# /lib/systemd/system/check_partitions@.service

# where <device> is the escaped device name. For example, if your device is
# /dev/bootflash, then <device> would be dev-bootflash

# All platforms in polaris share this file. Each platform has their own
# partitions_def.sh which is used by this script. partitions_def.sh defines
# the array "partitions" describing the partition table for the given device.
# Optionally, the platform specific file can define a "unit" variable that
# determines which units the partitions are defined in. Blocks the default
# if none is specified.
#
# See partitions_def.edison.sh for an example of a one of these files.
#

set -eu -o pipefail

# If the disk has existing partitions and we are potentially only
# resizing some of them when we create the new partition list
# then we cannot allow the combined fsck/mkfs step that comes after us
# to fixup the filesystem with an fsck.  This is because we may end
# up with a perfectly valid filesystem that does not include all
# of the blocks in the partition. fsck.ext2 does not complain in this
# situation if the partition is big enough, even if the first superblock
# is zeroed out.  For multi-gig partitions it takes some time to zero out
# the entire partition, so we only try to zero out all of the potential
# ext2 superblock locations instead.
function trash_ext2_superblocks {
    local go_on=1
    local start_offset=0
    local this_partition=$1

# Minimum spacing between ext2 superblocks is something like 8192
# 1024-byte blocks.  Spacing can be larger than this if ext2
# blocksize is larger, but we aren't going to be able to figure
# out the correct blocksize here.  So assume a worst case.
    set +e
    while [[ $go_on -ne 0 ]]; do
        # dd emits too many chatty messages, so suppress them
        dd if=/dev/zero of=${this_partition} bs=4096 count=1 seek=${start_offset} >/dev/null 2>&1
        if [[ $? -ne 0 ]]; then
            go_on=0
        else
            # 8192 1024-byte blocks is 2048 4096-byte blocks
            start_offset=$(( ${start_offset} + 2048 ))
        fi
    done
    set -e
}
readonly -f trash_ext2_superblocks

if [[ ! -e /etc/partitions_def.sh ]]; then
    echo "No PD partitions definition. Skipping partitions check"
    exit 0
fi

dev=$1

echo "Verifying partition table for device $dev..."

# "partitions" array defined in here
source /etc/partitions_def.sh

# assume MBR is the default partition type
if [[ ! -v partition_table_type ]]; then
    partition_table_type=MBR
fi

# to verify whether the partition table is correct, we just check that the
# total number of partitions is what it's supposed to be.
# While checking the size would seem reasonable, this causes other issues
# such as descrepancies in the partition start address and location due to
# alignment constraints and variations in hardware.

if [ $partition_table_type == "GPT" ]; then
    echo "GPT Partition detected"
    if [ -v 'unit' ]; then
        echo "unit variable is set for partition type GPT - exit"
        exit 1
    fi

    actual_partition_count=$(sgdisk -p "$dev" | awk 'BEGIN {lc=0; accum=0;}
                                                     /^Number +Start/{accum=1;}
                                                     /^ *[0-9]+ /{if(accum)lc++;}
                                                     END {print lc;}')

    if [[ ${#partitions[@]} == $actual_partition_count ]]; then
        echo "The partition table for device $dev appears to be correct."
        echo "No further action required"
    else 
        echo "The partition table for device $dev appears to be incorrect" >&2
        echo "The number of partitions should be ${#partitions[@]}, but was found to be $actual_partition_count" >&2
        echo "Rewriting the partition table..." >&2
        # clear the existing partitions
        # Use --zap-all instead of --clear since --clear will fail if
        # the existing partition table is damaged
        sgdisk --zap-all $dev >&2
        # create the partitions
        for i in "${!partitions[@]}"
        do
          sgdisk -n 0:0:+${partitions[$i]} -t 0:8300 -c 0:"$dev$i" $dev
        done
        sync

        # zero out potential ext2 superblocks to try to force mkfs
        # to happen instead of fsck.
        for i in "${!partitions[@]}"
        do
          trash_ext2_superblocks $dev$i
        done
        sync
    fi
else
    set +e
    # grep returns an exit status of 1 when there are no matches
    actual_partition_count=$(sfdisk -l "$dev" | grep -c "^$dev")
    set -e

    BOARD_TYPE=$( cat /sys/bus/platform/devices/cpld/board_subtype )

    partition_number_correct=0
    partitions_size_correct=1

    if [[ ${#partitions[@]} == $actual_partition_count ]]; then
        echo "The number of partition table appears to be correct."
        partition_number_correct=1

        if [[ $BOARD_TYPE == "SPARROW" ]]; then
            # if partition further check size is needed, do so
            check_func=`declare -f -F partitions_size_check`
            if [ $check_func == "partitions_size_check" ]; then
                partitions_size_correct=`partitions_size_check`
            fi
        fi
    fi

    if [[ $partition_number_correct -ne 1 ]] || [[ $partitions_size_correct -ne 1 ]]; then
        echo "The partition table for device $dev appears to be incorrect" >&2
        echo "The number of partitions should be ${#partitions[@]}, but was found to be $actual_partition_count" >&2
                
        if [[ $BOARD_TYPE == "HOTSPRINGS2" || $BOARD_TYPE == "STRIKER_O" || $BOARD_TYPE == "STRIKER_I" || $BOARD_TYPE == "STRIKER_C" || $BOARD_TYPE == "CRETE" || $BOARD_TYPE == "HOTSPRINGS" || $BOARD_TYPE == "PEG1M" ]]; then
            echo "Zeroing few blocks of ${dev}" >&2
            dd if=/dev/zero of=${dev} bs=128K count=400
        fi

        echo "Rewriting the partition table..." >&2

        if [[ $partitions_size_correct -ne 1 ]]; then
            oper="resize"
        else
            oper=""
        fi

        if [[ ! -e /etc/partitions_format.sh ]]; then
            printf ',%s\n' "${partitions[@]}" | sfdisk -u"${unit:-B}" -f "$dev"
        else
            /etc/partitions_format.sh $oper
        fi
        echo "The partition table was succesfully written" >&2
    fi
fi
