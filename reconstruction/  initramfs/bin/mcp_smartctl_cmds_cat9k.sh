#!/bin/bash
# SMART Deamon Script
#
# Jan 2018, Nikhil Aggarwal
#
# Copyright (c) 2013-2014, 2016-2018 by cisco Systems, Inc.
# All rights reserved
#

source /common

readonly SMART_HEALTH_CHECK_MSG="PASSED"
readonly SMART_ERROR_LOG_CHECK_MSG="No Errors Logged"
readonly SMART_ERROR_REPORT_FILE="/crashinfo/tracelogs/smart_errors.log"
readonly SMART_OVERALL_HEALTH_REPORT_FILE="/flash/smart_overall_health.log"
readonly SMART_ERROR_REPORT_CMD="please do 'more crashinfo/tracelogs:smart_errors.log'." 
readonly SMART_SELF_TEST_TIME_STR="Test will complete after "
readonly SMART_SELF_TEST_PROGRESS_STR="Self-test routine in progress"
readonly SMART_SELF_TEST_ABORTED="Aborted by host"
readonly SMART_SELF_TEST_INTERRUPTED="Interrupted (host reset)"
readonly SMART_SELF_TEST_SUCCESS_STR="Completed without error"
readonly ERR_MSG="Info:Unable to start SMART deamon";
readonly SMARTCTL_PATH="/usr/sbin/smartctl" 
readonly SMART_ENABLE_MSG="SMART Attribute Autosave Enabled."
readonly SMART_PID_FILE="/tmp/smart.pid"
readonly PV_INFO="/tmp/pv_mount_info"

#selft test intervals in days,
#Following is the schedule of different SMART tests relative to time of insertion
# 1. Offline Test - Every Two Days
# 2. Short Test   - Every Six Days
# 3. Extended Test - Every Fourteen Days
readonly SMART_SELF_TEST_INTERVAL=2
#This is the array of supported disk models for this script.
#To add this arry: Ex: SUPPORTED_MODELS=("MicroP400m" "New-Model")
#Allow all devices
readonly SUPPORTED_MODELS=("INTELSSDSCK*" "H9M2S86Q120GLE3G" "Micron5100MTFDD*" ".*")

#e.g. /dev/usb11 for Nyquist and /dev/disk01 for Macallan
RAID_DISK_PATH=$(cat /tmp/iHD-present | grep devpath: | cut -d : -f 2)
disk_health_fail=0
smart_logs_fail=0
offline_test_fail=0

# We only run SMART on supported models of the disk.
function check_supported_models(){
    local disk_model=$(/sbin/hdparm -I $1 | grep -o "Model Number:.*"|sed 's/[ _]//g' | cut -d: -f 2);
    local check=""
    for mdl in "${SUPPORTED_MODELS[@]}"
    do
        check=$(echo "$disk_model" | grep "$mdl")
        if [ ! -z "$check" ]; then
            is_supported=1
            return
        fi
    done
    is_supported=0
}

#We need to get the device path of the NIM-SSD Disks. 
#TODO  Currently we are making use of fact that only one NIM-SSD disk
# is allowed in a chassis. Basing on this assumption we get the kernel device names
# for the NIM-SSD drives from RAID_DISK_PATH.
function get_disk_dev_path(){
    for dev_path in $(ls $RAID_DISK_PATH 2>/dev/null); do
        if [[  $dev_path =~ "lvm" ]]; then
            pvdisplay -m > $PV_INFO
            dev_path=$(awk '/PV Name/ {print $3}' $PV_INFO)
        fi
        if [  -z "$disk1_dev_path" ]; then
            disk1_dev_path=$dev_path;
            # Check if device is present or not before proceeding. 
            if [ ! -b $disk1_dev_path ]; then
               echo "USB3.0 is not present, nothing to panic - This is after a reset" >> /tmp/usb3.0pwr_reset.log
               # Always maintain the latest log in flash, so delete and copy the log file to flash 
               /bin/rm -rf /flash/usb3.0pwr_reset.log
               /bin/cp /tmp/usb3.0pwr_reset.log /flash/usb3.0pwr_reset.log
               exit
            fi
            # Now I check if the disks are supported,if not I zero out their 
            # dev path, which would skip SMART stuff running on them.
            check_supported_models $disk1_dev_path;
            if [ $is_supported -eq 0 ];then
                disk1_dev_path="";
                exit 1
            fi
        else 
            disk2_dev_path=$dev_path;
            check_supported_models $disk2_dev_path;
            if [ $is_supported -eq 0 ];then
                disk2_dev_path="";
            fi
        fi;
    done;
}

#We need to enable SMART attributes on the drive everytime
# just to make sure they are enabled.
function enable_smart_attr(){
    local err=""
    if [ ! -z $disk1_dev_path ]; then
        err=$($SMARTCTL_PATH -d sat -S on $disk1_dev_path | grep "$SMART_ENABLE_MSG")
        if [ -z "$err" ]; then
            echo "INFO:Cannot Enable SMART on DISK" >> $SMART_ERROR_REPORT_FILE
        fi
    fi
    if [ ! -z $disk2_dev_path ]; then
        err=$($SMARTCTL_PATH -d sat -S on $disk2_dev_path | grep "$SMART_ENABLE_MSG")
        if [ -z "$err" ]; then
            echo "INFO:Cannot Enable SMART on DISK" >> $SMART_ERROR_REPORT_FILE
        fi
    fi
}

#Check the health of the disk and report if errors
function check_disk_health() {
    disk_health_fail=0;
    disk_health_check=$($SMARTCTL_PATH -d sat -H $1 | grep "$SMART_HEALTH_CHECK_MSG");
    # Always overwrite the health file as we always want to keep the latest log. 
    echo "$($SMARTCTL_PATH -d sat -H $1)" > $SMART_OVERALL_HEALTH_REPORT_FILE
    if [ -z "$disk_health_check" ]; then
        disk_health_fail=1
    fi
}

#Check for smart error logs
function check_smart_err_logs(){
    smart_logs_fail=0;
    smart_logs_check=$($SMARTCTL_PATH -d sat -l error $1 | grep "$SMART_ERROR_LOG_CHECK_MSG")
    if [ -z "$smart_logs_check" ]; then
        smart_logs_fail=1;
    fi
}

#Generate report if any the tests detect any failures should always be called after test functions.
#This function checks all the test flags and generates a report if atleast one of them fails and also 
#posts a log message on IOS.
function generate_report(){
    #check if health status repored any errors
    if [ $disk_health_fail -eq 1 -o $smart_logs_fail -eq 1 -o $offline_test_fail -eq 1 ]; then 
        echo "===============================START Logs for $1 on "$(date)"===================================" >> $SMART_ERROR_REPORT_FILE
    fi
    if [ $disk_health_fail  -eq 1 ]; then
        dual_echo SMART_LOG "ERR:$1:Health check failed:$SMART_ERROR_REPORT_CMD"
        echo "=====================Disk Health Report=====================" >> $SMART_ERROR_REPORT_FILE;
        $SMARTCTL_PATH -d sat -H $1 >> $SMART_ERROR_REPORT_FILE;
    fi
    if [ $smart_logs_fail -eq 1 ]; then
        #Smart error logs are not so catastrophic, hence INFO
        dual_echo SMART_LOG "INFO:$1:SMART error present:$SMART_ERROR_REPORT_CMD"
        echo "=====================SMART LOGS ERROR=====================">> $SMART_ERROR_REPORT_FILE;
        $SMARTCTL_PATH -d sat -l error $1 >> $SMART_ERROR_REPORT_FILE;
    fi
    if [ $offline_test_fail -eq 1 ];then
        dual_echo SMART_LOG "ERR:SMART OFFLINE test failed.:$SMART_ERROR_REPORT_CMD"
        echo "=====================Offline Self-test Results=====================">> $SMART_ERROR_REPORT_FILE;
        $SMARTCTL_PATH -d sat -l selftest $1 >> $SMART_ERROR_REPORT_FILE;
    fi
    if [ $disk_health_fail -eq 1 -o $smart_logs_fail -eq 1 -o $offline_test_fail -eq 1 ]; then 
        echo "=====================SMART DUMP=====================" >> $SMART_ERROR_REPORT_FILE;
        $SMARTCTL_PATH -d sat -a $1 >> $SMART_ERROR_REPORT_FILE;
        echo "===============================END Logs for $1 on "$(date)"===================================" >> $SMART_ERROR_REPORT_FILE
    fi

}


#Executes various smart tests on both drives and if error occurs creates a report
function execute_smart_tests(){
    if [ ! -z $disk1_dev_path ]; then
        check_disk_health $disk1_dev_path
        check_smart_err_logs $disk1_dev_path
        generate_report $disk1_dev_path
    fi
    if [ ! -z $disk2_dev_path ];then
        check_disk_health $disk2_dev_path
        check_smart_err_logs $disk2_dev_path
        generate_report $disk2_dev_path
    fi
}

#check_selftest_log arg1:device arg2:string to match
function check_selftest_log(){
    echo $($SMARTCTL_PATH -d sat -l selftest $1 | grep -i -o "# 1 .*"| grep -o "$2")
}

#Ignore vendor specific values - required for Nyquist SSD as suggested by Vendor. 
#Following values are ignored
#"0x80\|0x31\|0x4a\|0xe6\|0x72\|0xe6\|0x5b\|0x7a\|0xe3\|0xbb\|0x10\|0x3d\|0x55\|0xc1\|0x40\|0x19\|0x67\|0x48\|0xf4\|0x8c”
function ignore_vendor_specific_values_notsupported(){
    echo $($SMARTCTL_PATH -d sat -l selftest $1 | grep -i -o "# 1 .*" | grep "servo\|seek failure\|Unknown status\|unknown failure\|electrical failure\|read failure\|Interrupted\|handling damage\|Aborted by host")
}

#Schedules SMART test and waits for the result to complete before returning.
#After scheduling it sleeps of the time returned by smarctl -t command.
#And then checks the output of smartctl -l selftest and waits until the test is in complete state.
#We have to do this as sometimes it takes more time than the one returned by smartctl -t command.
#Nyuist SSD need additionl option in smartctl i.e. -d sat 
function schedule_self_test(){
    offline_test_fail=0
    local completion_time=$($SMARTCTL_PATH -d sat -t $2 $1 | grep "$SMART_SELF_TEST_TIME_STR" | sed 's/Test will complete after //')
    local completion_time_secs=$(date +%s -d "$completion_time")
    local cur_time=$(date +%s)
    local sleep_secs=$(expr $completion_time_secs - $cur_time)
    sleep $sleep_secs
    #Checks if the Test is still in progress.
    local check=$(check_selftest_log "$1" "$SMART_SELF_TEST_PROGRESS_STR")
    local smart_ignore_vendor_specific=$(ignore_vendor_specific_values_notsupported "$1")
    local test_abort_check=$(check_selftest_log "$1" "$SMART_SELF_TEST_ABORTED")
    local test_interrupt_check=$(check_selftest_log "$1" "$SMART_SELF_TEST_INTERRUPTED")
    while [ 1 ] ;do
        #if its not in progress state then we check if the result is "PASSED". Else if we mark it as Fail.
        if [ -z "$check" ];then
            check=$(check_selftest_log "$1" "$SMART_SELF_TEST_SUCCESS_STR")
            # Aborted by host and interrupted checks are required for pluggable devices
            # and should not be considered as test failures.
            if [[ -z "$check" && -z "$test_abort_check" && -z "$test_interrupt_check" && -z "$smart_ignore_vendor_specific" ]];then
                offline_test_fail=1;
            fi
            break;
        fi
        sleep 1
        check=$(check_selftest_log "$1" "$SMART_SELF_TEST_PROGRESS_STR")
        echo $check
    done
    #After completion of test we check for all the SMART tests and look for errors.
    execute_smart_tests $1
}

#Starts the smart deamon which schedules various SMART self tests with frequency determing by SMART_SELF_TEST_INTERVAL 
#It sleeps SMART_SELF_TEST_INTERVAL days and on every 3*SMART_SELF_TEST_INTERVAL executes Short test and 7*3*SMART_SELF_TEST_INTERVAL 
# executes Extended test.
function start_smart_deamon(){
    local count=0;
    local sleep_interval=$(expr $SMART_SELF_TEST_INTERVAL \* 3600 \* 24 )
    echo $BASHPID >$SMART_PID_FILE
    while [ 1 ]; do
        if [ ! -z $disk1_dev_path ]; then
            case "$count" in 
                3)
                 schedule_self_test $disk1_dev_path "short"
                 ;;
                7)
                 schedule_self_test $disk1_dev_path "long"
                 ;;
                *)
                 schedule_self_test $disk1_dev_path "offline"
            esac
        fi
        if [ ! -z $disk2_dev_path ]; then
            case "$count" in 
                3)
                 schedule_self_test $disk2_dev_path "short"
                 ;;
                7)
                 schedule_self_test $disk2_dev_path "long"
                 ;;
                *)
                 schedule_self_test $disk2_dev_path "offline"
            esac
        fi
        count=$(expr $count + 1 )
        if [ $count -eq 7 ];then
            count=0
        fi
        #Sleep for SMART_SELF_TEST_INTERVAL days
        sleep $sleep_interval 
    done;
}

#Init functions to handle start and stop of the deamon
function init(){
    local tmp_model;
    if [ "$1" == "start" ]; then
        if [ ! -f $SMARTCTL_PATH ] ; then
            #can't find smartctl in $PATH
            echo "$(date) ""$ERR_MSG:'smartctl' not available" > $SMART_ERROR_REPORT_FILE
        else
            get_disk_dev_path
            enable_smart_attr
            execute_smart_tests
            dual_echo SMART_LOG "$(date) ""INFO: Starting SMART deamon" > $SMART_ERROR_REPORT_FILE
            #To avoid race condition touch PID file and declare to start a daemon
            touch $SMART_PID_FILE
            start_smart_deamon &
        fi
   elif [ "$1" == "stop" ]; then
       #We grep for script nane and kill the script
       pid=$(cat $SMART_PID_FILE);
       echo "$(date) ""INFO:Killing SMART monitor deamon($pid)." >> $SMART_ERROR_REPORT_FILE
       kill -KILL $pid
       rm -f $SMART_PID_FILE
   elif [ "$1" == "explode" ];then
       #Now we explode the required .pkg files in /firmware.
       local model=$(echo $2 | sed 's/[\ _]//g')
       local check=""
       check=$(file -L /firmware/$model.pkg | cut -d: -f2 | grep gzip)
       if [[ "$check" != "" ]]
           then
           /bin/tar -xvzf /firmware/$model.pkg -C /firmware &> /dev/null
       else
           echo "%NIM-XXD:INFO:Invalid package file.";
           exit 1;
       fi
   fi
}
init $1 $2 

