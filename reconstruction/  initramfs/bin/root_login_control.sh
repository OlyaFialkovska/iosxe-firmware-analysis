#!/bin/bash
#
# October 2016, Clinton Grant
#
# Copyright (c) 2016 by Cisco Systems, Inc.
# All rights reserved
#

#
# This script finds controls the enabling of root login for Development
# Mode. The root account remains locked in the field deployed devices.
#
# This script is dependent on the find_debug_conf.sh having run to process
# the DEBUG_CONF rommon variable.


# enable_root_login
#
# If the DEBUG_CONF rommon variable references a debug.conf file
# we enable the root account login by modifying the /etc/passwd file.
#
function enable_root_login
{
    # Check that the /tmp/debug.conf file is non-zero in size, as installed by
    # the find_debug_conf function.
    if [[ -s /tmp/debug.conf ]]; then
        sed -i 's/root:\*/root:CAuu7meecu9\/A/g' /etc/passwd
        echo "root account login ENABLED."
    fi
}
