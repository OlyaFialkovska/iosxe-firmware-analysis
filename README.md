# Firmware Image Analysis

## Objective

Given two firmware images, fully reconstruct the file directory structure by
understanding the underlying file format — not by scanning for magic bytes.

Write a Golang parser.

## Background

A naive approach (e.g. binwalk) scans firmware byte-by-byte looking for known
signatures. While useful for initial discovery, this doesn't reflect how a real
device boots: the bootloader already *knows* the layout because it understands
the file format and its structural metadata.

The goal here is to replicate that understanding.

## Requirements

1. **Parse the file format structurally.** Walk the structure as the device would.

2. **Fully reconstruct the directory tree.** Extract every file and directory
   contained in the firmware. The output should be a complete, browsable
   replica of the original filesystem.

3. **Account for all bytes.** Every region of the image should be explained —
   bootloader, kernel, filesystem(s), padding, checksums, unused space, etc.
   No unexplained gaps.

4. **Deeper binary analysis (bonus).** Disassembly, entropy analysis,
   compression identification, or other low-level investigation is welcome but
   not required.

## Guidelines

- **Avoid AI-assisted submissions.** The point is to build a deep, personal
  understanding of the firmware format. If you do use any AI-generated tools or
  prompts during your research, document them fully — include the exact prompts
  and explain how they fit into your workflow.

- **No binwalk-style extraction.** You may use binwalk (or similar) for initial
  reconnaissance, but your final extraction must be driven by parsing the
  format's own structural metadata.

## Deliverables

- Extracted directory tree for at least 1 firmware image.
- A write-up explaining the file format structure, how you parsed it, and how
  every byte range in the image is accounted for.
- Any scripts or tooling you wrote to perform the extraction.
- (If applicable) Documentation of any AI tool usage, including prompts.
