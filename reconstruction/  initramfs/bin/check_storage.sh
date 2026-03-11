#!/bin/bash
#
# Copyright (c) 2018-2019 by Cisco Systems, Inc.
# All rights reserved.

source /common

cpld_path="${BINOS_CPLD_ROOT:=/tmp/chassis/cpld/chasfs/}"

# while we need to know our platform, we need to determine the
BOARD_SUBTYPE=$( cat $cpld_path/board_subtype )
BOARD_TYPE=$( cat $cpld_path/board_type )
FRU=${BOARD_TYPE,,}

check_disk $1
