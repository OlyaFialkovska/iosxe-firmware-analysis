# Commands Used During Firmware Analysis

This document lists the commands executed during the technical analysis of Cisco IOS-XE firmware images.

Target firmware images:

C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.06.SPA.bin  
C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.07.SPA.bin


Initial File Inspection

file C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.07.SPA.bin


Firmware Structure Discovery

binwalk C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.07.SPA.bin


Initial Extraction Attempt

binwalk -e C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.07.SPA.bin


Initramfs Validation

file decompressed.bin


Initramfs Extraction

mkdir initramfs_root  
cd initramfs_root  
cpio -idmv < ../decompressed.bin


Initramfs Metadata Discovery

grep -R "RELVER" .  
grep -R "Gibraltar" .


Kernel Version Discovery

ls lib/modules  
ls lib/modules/4.4.202


Linux Virtual Machine Environment Preparation

sudo apt update  
sudo apt install binwalk squashfs-tools


Mount Host Firmware Repository

sudo mkdir /mnt/share  
sudo mount -t 9p -o trans=virtio,version=9p2000.L share /mnt/share  
ls /mnt/share


Full Firmware Extraction (16.12.07)

mkdir ~/extraction_07  
cd ~/extraction_07  
binwalk -Me /mnt/share/C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.07.SPA.bin


Full Firmware Extraction (16.12.06)

mkdir ~/extraction_06  
cd ~/extraction_06  
binwalk -Me /mnt/share/C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.06.SPA.bin


Extracted Package Inspection

ls  
find . -name "*.pkg"


Active Package Configuration Inspection

cat packages.conf


Cross-Version Package Comparison

diff packages.conf ../extraction_06/packages.conf


Strings-based Network Indicator Analysis

strings -a -n 6 cat9k-rpbase.16.12.06.SPA.pkg | grep -i -m 30 "ssh"

strings -a -n 6 cat9k-rpbase.16.12.07.SPA.pkg | grep -i -m 30 "ssh"


Network Primitive Discovery

strings *.pkg | grep -i -E -m 30 "socket|bind|connect|ssl"


Package File Type Identification

file cat9k-rpbase.16.12.06.SPA.pkg  
file cat9k-rpbase.16.12.07.SPA.pkg


Secrets and Credential Artifact Scan

grep -R "BEGIN RSA PRIVATE KEY" .  
grep -R "BEGIN OPENSSH PRIVATE KEY" .  
grep -R "BEGIN CERTIFICATE" .  
grep -Ri "password" .  
grep -Ri "token" .


Network Artifact Discovery

grep -Ri "http" .  
grep -E -R "[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+" .


Web Interface Discovery

grep -Ri webui .


Secure Remote Access Discovery

grep -Ri "ssh" .


Hex Inspection of .pkg Container

xxd -g 1 -l 512 cat9k-rpbase.16.12.06.SPA.pkg


Entropy Analysis

binwalk -E cat9k-rpbase.16.12.06.SPA.pkg


Binary Comparison Between Firmware Builds

cmp -l cat9k-rpbase.16.12.06.SPA.pkg cat9k-rpbase.16.12.07.SPA.pkg


ELF Binary Inspection

readelf -h <binary_file>


Binary Networking Indicator Discovery

strings -a -n 6 <binary_file> | grep -i -E "socket|connect|rpc"


Manual Payload Carving

dd if=cat9k-rpbase.16.12.07.SPA.pkg of=carved_segment.bin bs=1 skip=<offset> count=<size>


Carved Artifact Inspection

binwalk carved_segment.bin


Compression Header Inspection

xxd -g 1 carved_fragment.bin | head


Compression Validation

gunzip -c carved_fragment.bin


Custom Go Parser Execution

go build pkgscan.go

./pkgscan -in cat9k-rpbase.16.12.06.SPA.pkg
