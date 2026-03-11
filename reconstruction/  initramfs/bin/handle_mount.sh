#!/bin/bash
#
# Copyright (c) 2016-2018 by Cisco Systems, Inc.
# All rights reserved
#
# This file is used to do any work needed after a partition is mounted.
# It is lauched by the handle_mount@.service systemd service.
#
# To have this script run after a partition is mounted, you need to add
# a symlink from
# /lib/systemd/system/local-fs.target.wants/handle_mount@<mount_point>.service
# pointing to
# /lib/systemd/system/handle_mount@.service
#
# where <mount_point> is the escaped mount point path. For example, if your
# mount point is /mnt/sd1, then <mount_point> would be mnt-sd1
#
#
# All platforms in polaris share this file. They use this file in conjunction
# with their handle_mount${PD}.sh which each platform family defines.
# This file does the common task or creating directories on the mounted
# partition and create a udev-info file for IOS.
#
# Each platform family defines which directories to create and what values
# the udev-info file should contain. This is accomplished by each platform
# setting variables in their handle_mount${PD}.sh which are used by this file.
# The list of variables than need to be set can be found in the comment below.
#
# The platforms also have the option of setting pre and post hooks to do any
# platform specific tasks either before or after the directories are made and
# the udev-info file is created.
#
# See the handle_mount-ngwc.sh file for a good example of a
# handle_mount${PD}.sh file.
#


udev_dir=/tmp/udev/etc/udev

# platform specific script needs the mount point
mount_point=$2

# the platform specific script needs to set the following functions:
#      * pd_pre_hook
#      * pd_post_hook
# and it needs to set the following variables:
#      * device
#      * ios_name
#      * ios_root
#      * display_name
#      * ifs_type
#      * ifs_flags
#      * ifs_feature
#      * priority
#      * hidden
#
#      * dir_list

source handle_mount_pd.sh

function make_dirs
{
    for dir_name in $dir_list; do
        mkdir -p -m 0755 "$mount_point"/"$dir_name"
    done
}

function write_udev_info
{
    fstype=$( /usr/bin/file -sL $device | /usr/bin/awk '{print $5}' )
    # blockdev returns the device size in bytes just like what "fdisk -s" 
    # returns.  Need to divide by 1024 to convert to Kbytes since 
    # 'show version' output for flash device sizes is in Kbytes.
    partsize=$( blockdev --getsize64 $device )
    partsize=$(( $partsize / 1024 ))
    # extract partition name to make unique file name, otherwise
    # multiple handle mount processes operating same file
    # will corrupt the temp file
    partn=$(echo "$mount_point" | sed -e 's/.*\///')
    tmpinfo=/tmp/info_file$partn

    printf  "devname:%s\n"      "$ios_name"     >   $tmpinfo
    printf  "mntpath:%s\n"      "$ios_root"     >>  $tmpinfo
    printf  "devpath:%s\n"      "$device"       >>  $tmpinfo
    printf  "fstype:%s\n"       "$fstype"       >>  $tmpinfo
    printf  "partsize:%s\n"     "$partsize"     >>  $tmpinfo
    printf  "displayname:%s\n"  "$display_name" >>  $tmpinfo
    printf  "priority:%s\n"     "$priority"     >>  $tmpinfo
    printf  "ifstype:%s\n"      "$ifs_type"     >>  $tmpinfo
    printf  "ifsflags:%s\n"     "$ifs_flags"    >>  $tmpinfo
    printf  "ifsfeature:%s\n"   "$ifs_feature"  >>  $tmpinfo
    printf  "hidden:%s\n"       "$hidden"       >>  $tmpinfo
    if [[ "$serial_nm" != "" ]]; then
        printf "serialnm:%s"     "$serial_nm"   >> $tmpinfo
    fi

    /bin/mkdir -Z -p "$udev_dir/$ios_name"
    udev_info="$udev_dir/$ios_name/udev-info"

    # write udev info to a temp file first since an event is created
    # for any changes in the udev directory. this allows all data to
    # be written at once and only generates one event, see CSCvf53751.
    
    /bin/cp -Z $tmpinfo $udev_info
     
    # uncomment for debug
    # echo $udev_info >> $tmpinfo
    # echo $mount_point >> $tmpinfo
    # echo partn $partn >> $tmpinfo

    /bin/rm $tmpinfo
}

function start
{
    pd_pre_hook
    make_dirs
    write_udev_info
    pd_post_hook
    declare -f -F pd_start > /dev/null
    if [[ $? == 0 ]]; then
        pd_start $ios_name
    fi
}

function stop
{
    rm -rf "$udev_dir/$ios_name/"
    declare -f -F pd_stop > /dev/null
    if [[ $? == 0 ]]; then
        pd_stop $ios_name
    fi
}

case "$1" in
    start|stop) "$1" ;;
    *)
        echo "Usage: $0 {start|stop}"
        exit 1
esac
