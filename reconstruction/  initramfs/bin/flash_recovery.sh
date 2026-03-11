#!/bin/bash
#
# Flash Recovery script
# Run the flash-rec binary and if the errors are found, power reset the switch
#
# Copyright (c) 2020 by Cisco Systems, Inc.
# All rights reserved.
#

LOGFILE="/tmp/flash_recovery.txt" 
DIR_PATH="/tmp/sw/rp/0/0/rp_base/mount/usr/platform/"
readonly SCRIPTS=/etc/init.d
source $SCRIPTS/cat9k_fpgahelper_funcs.sh
PWRCYCLE_COOKIE="/tmp/flash_rec-cookie.sh"

echo "Run the Flash recovery binary" >> $LOGFILE 2>&1
# Run the binary
/usr/bin/flash-rec

# Additional check before proceeding with the power reset 
# Check if kmsg file has the switch reset message written by flash-rec code
if grep -aq "Detected Flash issues" /dev/kmsg ; then
   echo "Flash errors found, proceeding with the logic to power cycle the switch" >> $LOGFILE 2>&1

   # Call the fpga helper script to read fpga_image_descr info
   # used to extract power cycle device 
   if [[ -f /tmp/platform_info/C9300_INFO ]]; then
       # this nyquist classic
       PLATFORM_TYPE="NyquistClassic"
       FPGA_IMAGE_DESCR="polaris_cat9k_strutt_fpga_descr.txt"
       ret_val=1
   else 
       fpga_helper_set_env_var
       ret_val=$?
   fi
   if [ $ret_val -eq 1 ]; then
      echo "$(date) env set. starting power reset" >> $LOGFILE 2>&1
      for descr in $FPGA_IMAGE_DESCR; do

          # build the power cycle string for this device from the product template string
          # the place holder "PCI_DEVICE" gets replaced with the real pci device number

          PWR_DEVICE_NUM=$(grep pwrcycleDev: $DIR_PATH/$descr | cut -f 2 -d ':' )
          PWR_PCI_DEVICE=`lspci -D | grep $PWR_DEVICE_NUM | cut -c1-12`
          PWR_CYCLE_SCRIPT=$(grep pwrCycle: $DIR_PATH/$descr | cut -f 2 -d ':' )
          DevPwrCycle="$(echo $PWR_CYCLE_SCRIPT | sed "s@PCI_DEVICE@$PWR_PCI_DEVICE@")"
          echo $DevPwrCycle >> $LOGFILE 2>&1

          # Sleep required to see the message on the console before Power cycle
          sleep 10 
          # set the cookie and use it in the reload sequence
          echo "#!/bin/bash"    > $PWRCYCLE_COOKIE
          echo "sync"          >> $PWRCYCLE_COOKIE
          echo -e $DevPwrCycle >> $PWRCYCLE_COOKIE
          chmod 777               $PWRCYCLE_COOKIE
      done
   fi

   # display the console message continuosly every 1 hour when the flash issue is seen.
   while true; do
      echo "<3> *** Detected Flash issues. Please issue the reload slot <#> command at your earliest convenience ***" > /dev/kmsg
      sleep 1h;
   done
fi   
