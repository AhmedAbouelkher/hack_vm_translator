# VM Translator - Project 7 (Nand2Tetris)

This is a VM (Virtual Machine) translator for **Project 7** of the [Nand2Tetris course](https://www.nand2tetris.org/).

## Overview

The VM translator converts VM code (`.vm` files) to Hack assembly code (`.asm` files) that can run on the Hack computer platform.

## Current Status

⚠️ **This translator is not finished** - it's a work in progress.

## Usage

### Basic Usage

```bash
go run main.go -s <source.vm> [-o <output.asm>] [-c <compare.asm>]
```

### Examples

```bash
# Convert a VM file to assembly
go run main.go -s Add.vm

# Specify output file
go run main.go -s Add.vm -o Add.asm

# Compare with expected output
go run main.go -s Add.vm -c expected.asm
```

### Batch Processing

```bash
# Process all .vm files in current directory
./run_all_vm.sh

# Clean up generated .asm files
./clean_asm.sh
```

## Supported Commands

The translator currently supports:

- **Push/Pop operations** for various memory segments
- **Arithmetic operations** (add, sub, neg, eq, gt, lt, and, or, not)

## Project Structure

- `main.go` - Main translator implementation
- `run_all_vm.sh` - Script to process all .vm files
- `clean_asm.sh` - Script to remove generated .asm files

## Nand2Tetris Course

This project is part of the Nand2Tetris course, which teaches how to build a complete computer system from the ground up, starting with NAND gates and ending with a high-level programming language.
