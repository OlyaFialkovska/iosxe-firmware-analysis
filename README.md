# Cisco IOS-XE Firmware Analysis

This repository contains the results of a technical analysis of Cisco IOS-XE firmware images performed as part of an internship technical assignment.

## Analyzed firmware

C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.06.SPA.bin  
C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.07.SPA.bin

## Analysis goals

- Investigate internal firmware structure
- Extract embedded components
- Compare two firmware builds
- Analyze proprietary `.pkg` container format

## Tools used

- file
- binwalk
- strings
- grep
- unsquashfs
- xxd
- custom Go-based signature scanner

## Key findings

The firmware follows a layered architecture:
