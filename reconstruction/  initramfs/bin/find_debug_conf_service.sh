#!/bin/bash
#
# October 2016, Clinton Grant
#
# Copyright (c) 2016 by Cisco Systems, Inc.
# All rights reserved
#

#
# This script is a wrapper executed by the find_debugconf SystemD service.
source /bin/find_debug_conf.sh
source /bin/root_login_control.sh

enable_root_login

