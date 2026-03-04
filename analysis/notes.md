# notes.md — Technical Notes (Firmware Analysis Week)

Author: Fialkovska Olya  
Scope: Cisco IOS-XE Cat9k firmware analysis  
Targets:
- C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.06.SPA.bin
- C9200_9300_9400_9500_9600_cat9k_iosxe.16.12.07.SPA.bin

---

## 1) High-Level Structure (binwalk results)

The firmware image is a multi-layer container. Binwalk identified:

- Embedded **GZIP** segment at offset **0x487B** (dec 18555).
- Embedded **initramfs** archive at offset **0x4E9A20** (dec 5151264), named:
  `initramfs.x86_64.cat9k.ramfs.cpio` (CPIO, x86_64).
- A large **SquashFS** filesystem (v4.0, XZ compressed, ~782MB) serving as the main runtime layer.

Interpretation:
This is consistent with a Linux-based boot workflow:
- initramfs = early boot environment
- SquashFS = runtime container for IOS-XE modular packages

---

## 2) Extraction Workflow and Environment Constraint

Initial extraction attempt on macOS:

- `binwalk -e` extracted the GZIP segment successfully.
- SquashFS extraction failed due to missing external extractor tools on macOS (sasquatch / compatible unsquashfs flow not available).

Resolution:
A Linux VM was deployed (UTM on Apple Silicon) to complete extraction using native tooling.

Linux VM setup:
- Ubuntu 22.04 ARM64
- Tools installed:
  - binwalk
  - squashfs-tools

Host → VM file access:
- Host firmware repository mounted into Linux via VirtFS (9P) at:
  `/mnt/share`

---

## 3) Initramfs Layer Findings (Boot Layer)

The extracted initramfs payload (`decompressed.bin`) was confirmed as:

- `ASCII cpio archive (SVR4 with no CRC)`

Filesystem layout was consistent with a minimal Linux boot environment:
- `bin/`, `etc/`, `init`, `lib/`, `lib64/`, `usr/`, `sbin/`, `dev/`, `sys/`, ...

Boot behavior indicators:

- `/init` present → boot entry point.
- Cisco-specific scripts observed:
  - `mount_packages.sh`
  - `verify_packages.sh`
  - `rommon_vars.sh`

Boot script logic (key points):

- `verify_packages.sh`
  - verifies packages using:
    `verify_pkg --bootverify <package>`
  - indicates packages are digitally signed and validated during boot

- `mount_packages.sh`
  - prepares/mounts runtime packages
  - uses rsync synchronization
  - invokes install engine:
    `install_engine.sh --operation install_fp_cc_boot`

- `rommon_vars.sh`
  - imports ROMMON environment variables
  - sets system parameters prior to IOS-XE runtime startup

Interpretation:
Boot flow is modular + integrity-checked; package authenticity is validated before runtime launch.

---

## 4) Raw Firmware Fingerprint Discovery (Initramfs)

Version-identifying artifacts were extracted from initramfs strings/metadata:

- `RELVER=16.12.07`
- Build: `16.12.07`
- Full build string: `16.12.07.0.6565.1643829996`
- Codename: `Gibraltar`

Kernel layer indicator:

- kernel modules observed under:
  `lib/modules/4.4.202/`
→ Linux kernel version: **4.4.202**

Architecture indicator:

- initramfs filename includes:
  `initramfs.x86_64`
→ architecture: **x86_64**

These values form stable identifiers usable for deterministic fingerprinting.

---

## 5) Full Root Filesystem Extraction (SquashFS Layer)

Full extraction was performed in Linux using recursive binwalk:

- For 16.12.07:
  extraction directory: `~/extraction_07`
  input: `/mnt/share/...16.12.07...bin`

- For 16.12.06:
  extraction directory: `~/extraction_06`
  input: `/mnt/share/...16.12.06...bin`

SquashFS content structure (important observation):

- Not a standard Linux root filesystem.
- Contains modular IOS-XE packages (`*.pkg`) and `packages.conf`.

Examples observed (16.12.07):
- `cat9k-rpbase.16.12.07.SPA.pkg`
- `cat9k-webui.16.12.07.SPA.pkg`
- `cat9k-sipbase.16.12.07.SPA.pkg`
- `packages.conf`

Interpretation:
SquashFS acts as a container for IOS-XE modular packages rather than a monolithic rootfs.

---

## 6) packages.conf and Active Package Set

`packages.conf` confirms modular package mode:
- multiple functional components loaded independently

Key packages observed:
- rpbase
- webui
- sipbase
- srdriver
- guestshell
(and others like wlc, etc.)

Interpretation:
Web UI, routing services, drivers, and guest shell components are enabled through modular packaging.

---

## 7) Cross-Version Delta (16.12.06 vs 16.12.07)

A diff of `packages.conf` between versions showed:

- identical package structure
- identical module set
- no added/removed components
- only version identifiers updated (16.12.06 → 16.12.07)

Conclusion:
16.12.07 is consistent with a maintenance/patch-level update, not a structural redesign.

---

## 8) Extended Static Analysis (Network + Surface Indicators)

Strings-based SSH indicator scan on primary runtime package (`cat9k-rpbase.*.pkg`):

Observed SSH/NETCONF-related artifacts (both versions):
- `cli_ssh.beam`
- `confd_ssh.beam`
- `netconf_ssh`
- `netconf_server`
- `netconf_tcp`

Interpretation:
Embedded SSH service + NETCONF-over-SSH remote management support.

Network primitive keyword scan (socket/bind/connect/ssl):

Observed:
- `SSL`, `ssl`, and related SSL references
Interpretation:
Encrypted transport mechanisms exist in firmware; no anomalous networking keywords in initial triage.

---

## 9) Secrets / Credentials Triage

Goal:
Detect plaintext secrets and credentials in extracted layers.

Search patterns:
- RSA/OpenSSH private key blocks
- certificate blocks
- password/token keywords

Results:
- No plaintext private keys found.
- No `BEGIN CERTIFICATE` blocks found.
- No clear plaintext credentials from password/token keyword scans.

Interpretation:
No obvious plaintext secret artifacts were present in the extracted filesystem layer.

---

## 10) URLs and IPv4 Indicators

URL scan (`http`) and IPv4 regex scan detected matches primarily inside:
- compiled binaries
- compressed fragments
- embedded libraries

Examples observed:
- `http://www.pygkt.org`
- `http://www.freedesktop.org/standards/dbus`

Interpretation:
No explicit external endpoints or IPs were found as plaintext configuration values.
Matches likely originate from embedded libraries / compiled code strings.

---

## 11) Web Interface Evidence (webui)

Evidence via recursive search for `webui`:

Examples observed:
- `/usr/binos/conf/webui.conf`
- `/usr/binos/conf/webui-servers.conf`
- `/usr/binos/conf/webui_upgrade_helper.sh`
- `/usr/binos/conf/webui_smu_helper.sh`
- `/lib/tmpfiles.d/webui.conf`
- references to rp_webui package mounting

Interpretation:
Firmware includes integrated web management interface; nginx-related references suggest an embedded web server component.

---

## 12) SSH Service Evidence (system-level)

Evidence via recursive `ssh` search:

Examples observed:
- `/lib/systemd/system/sshd.service`
- `/etc/services: ssh 22/tcp`
- `/etc/services: ssh 22/udp`
- `/etc/cron: sshd`
- `/etc/avahi/services/ssh.service`

Interpretation:
SSHD is present, confirming secure remote administration capability.

---

## 13) Custom Go Parser (Signature Scanner)

Problem:
Cisco IOS-XE `.pkg` is not detected as a standard archive by `file` and is not trivially extractable by generic tooling.

Goal:
Detect embedded payload boundaries inside proprietary `.pkg` containers.

Approach:
A Go scanner was implemented to search for common magic bytes:
- GZIP: `1f 8b 08`
- XZ: `fd 37 7a 58 5a 00`
- SquashFS marker: `hsqs`
- ELF: `7f 45 4c 46`

Output:
- offsets for candidate embedded payloads
- data usable for manual carving and automated extraction

Structural conclusion:
`.pkg` behaves as a proprietary modular container embedding multiple compressed/structured segments, not a single archive.

---

## 14) Manual Payload Carving (cat9k-rpbase.16.12.07)

Using scanner offsets:
- SquashFS-related marker detected at:
  **0x1A742280**

Manual dd carving confirmed:
- region contains multiple XZ-compressed segments

Interpretation:
`.pkg` bundles multiple compressed payload blocks rather than a monolithic filesystem.

---

## 15) Automated Multi-Layer Extraction (pkgscan2)

Automation goal:
Increase coverage and repeatability across builds.

Workflow (validated on 16.12.06):
- detect signatures
- output JSON reports
- auto-carve segments into separate artifacts
- compute SHA256 for carved artifacts
- allow iterative triage (`binwalk carved/*.bin`)

Observed result:
- `.pkg` contains many distributed compressed modules (XZ/GZIP), not a single filesystem image.

Interpretation:
This supports a componentized packaging model: independently compressed runtime modules packaged inside a proprietary container.

---

## 16) Compression Header Anomaly (GZIP Method 236)

Observation:
Carved fragments contain a standard-looking GZIP magic header:
- `1f 8b 08`

But standard decompression failed.
`file` reported:
- `gzip compressed data, reserved method 236, encrypted`

Interpretation:
These GZIP-like segments are not standard DEFLATE streams (method 8).
They likely include proprietary wrapping/modification or encryption flags, reinforcing the need for format-aware parsing.

---

## 17) Firmware Fingerprint Framework (Structured View)

Stable identifiers across both builds:
- RELVER (version string)
- full build identifier (varies by release)
- codename: Gibraltar
- kernel: 4.4.202
- architecture: x86_64
- package filename suffixes reflecting release version

Delta summary (16.12.06 vs 16.12.07):
- updated version strings / build identifiers / filenames
- no observed changes in kernel, modular architecture, active package set, or key network indicators

---

## 18) ELF Triage (Binary-Level Check)

A statically linked x86-64 ELF executable carved/identified (associated with early offsets / ELF triage).

Header properties observed (readelf -h):
- ELF64, little endian
- System V ABI
- EXEC type
- x86-64 architecture
- statically linked
- entry point: 0x100000

Strings-based network indicators inside the ELF included:
- `hub_port_connect`
- `xs_local_setup_socket`
- `xs_tcp_setup_socket`
- `/var/run/rpcbind.sock`
- `TRACE_SYSTEM_SS_CONNECTING`
- `TRACE_SYSTEM_SS_CONNECTED`
- `TRACE_SYSTEM_SS_DISCONNECTING`

Interpretation:
Network interaction logic exists at the compiled binary layer (socket-level behavior), not only via high-level package metadata.

---

## 19) .pkg Deep Structure Notes (Header + TLV hypothesis)

Hex inspection of `.pkg` beginning showed ASCII-rich keys:

Examples:
- `KEY_TLV_PACKAGE_COMPATIBILITY`
- `KEY_TLV_PACKAGE_BOOTARCH`
- `KEY_TLV_BOARD_COMPAT`
- `KEY_TLV_CRYPTO_KEYSTRING`
- `CW_IMAGE`, `CW_VERSION`, `CW_FULL_VERSION`
- board/arch hints (e.g., `ARCH_i686_TYPE`, `FRU_RP_TYPE`)

Interpretation:
Header strongly suggests TLV-like metadata (Type–Length–Value) encoding:
- compatibility checks
- board constraints
- architecture requirements
- version identifiers
- cryptographic metadata

Entropy analysis:
- high entropy starting at offset 0x0 (≈0.999886)
→ most of the file is compressed/encrypted-like payload, with a smaller structured metadata region.

Alignment check:
- example payload offset 0x4E9A20 is not 4KB aligned (mod 4096 = 2592)
→ supports “structured container with internal offsets” rather than raw flash image alignment behavior.

---

## 20) Overall Conclusion (Notes-Level)

The firmware exhibits a layered architecture:

- initramfs boot layer with integrity-checked modular boot process
- SquashFS package container with modular IOS-XE components
- `.pkg` proprietary container embedding many compressed fragments
- non-standard compression markers (e.g., GZIP method 236) requiring format-aware parsing
- binary-level ELF artifacts confirming x86-64 userland components and networking logic

The workflow progressed from initial triage to deep container analysis and custom tooling, with cross-version comparison confirming 16.12.07 as a maintenance-level update relative to 16.12.06.
