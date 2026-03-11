#!/bin/bash

# mount_point is passed in

ifsflags_w_hidden=536871939
ifsflags_wo_hidden=1027

function pd_pre_hook
{
    if [[ "$mount_point" == "/mnt/sd3" ]]; then
        if [[ -L "$mount_point"/user ]]; then
            rm -f $mount_point/user
        fi
    fi
}

function pd_post_hook
{
    # export to UFS
    ufs_mopts="(rw)"
    ufs_eopts="(rw,no_subtree_check,insecure,no_root_squash,no_all_squash)"
    ufs_dir=/etc/conf/unifiedfs
    fs_list=$ufs_dir/PD_fslist
    mounts_conf=$ufs_dir/ng3k_mounts.conf
    exports_conf=$ufs_dir/ng3k_exports.conf

    ufs_mnt_pnt="/tmp/ufs/$ios_name"
    if [[ "$mount_point" == "/mnt/sd3" ]]; then
        root=$mount_point/user
    else
        root=$mount_point
    fi

    echo "$root $ufs_mnt_pnt $ufs_mopts" >> "$mounts_conf"
    echo "$root $ufs_mnt_pnt $ufs_eopts" >> "$exports_conf"
    echo "$ios_name" >> "$fs_list"

    # misc
    if [[ "$mount_point" == "/mnt/sd1" ]]; then
        # this file needs to be looked into
        if [[ -f /usr/binos/conf/ngd_read_startup_cfg.sh ]]; then
            source /usr/binos/conf/ngd_read_startup_cfg.sh
        fi
        # why do we touch this file? I can't find anything that reads it
        if [[ ! -f /crashinfo/koops.dat ]]; then
            touch /crashinfo/koops.dat
        fi
    fi

    /bin/rm -rf "$mount_point"/lost+found
}

#Partition 1, Primary, crashinfo:
#Partition 3, Primary, flash:
#Partition 5, Logical, lic0:
#Partition 6, Logical, lic1:
#Partition 7, Logical, obfl0:
#Partition 8, Logical, ucode0:
#Partition 9, Logical, drec0:

licensing_dirs=" \
    pri \
    red \
    eval \
    dyn_eval \
    persist/pri \
    persist/red \
    persist/pri_chk \
    persist/red_chk \
"

declare -A dir_lists=( \
    [1]= \
    [3]=".install user" \
    [5]=${licensing_dirs} \
    [6]=${licensing_dirs} \
    [7]= \
    [8]= \
    [9]= \
)

declare -A display_names=( \
    [1]="Crash Files" \
    [3]="Flash" \
    [5]="Licensing" \
    [6]="Licensing Backup" \
    [7]="Onboard Failure Logging" \
    [8]="Silent Roll" \
    [9]="Disaster Recovery" \
)

declare -A ios_names=( \
    [1]="crashinfo" \
    [3]="flash" \
    [5]="lic0" \
    [6]="lic1" \
    [7]="obfl0" \
    [8]="ucode0" \
    [9]="drec0" \
)

# partition number
pn=${mount_point: -1}

if [[ $pn == 3 ]]; then
    ios_root=$mount_point/user
else
    ios_root=$mount_point
fi

# expose for udev-info
device="/dev/bootflash${pn}"
ios_name=${ios_names[$pn]}
display_name=${display_names[$pn]}
ifs_type=64

if [[ $pn == 3 || $pn == 1 ]]; then
    ifs_flags=$ifsflags_wo_hidden
else
    ifs_flags=$ifsflags_w_hidden
fi

ifs_feature=84833

if [[ $pn == 3 ]]; then
    priority=1
else
    priority=2
fi

hidden=n

dir_list=${dir_lists[$pn]}
