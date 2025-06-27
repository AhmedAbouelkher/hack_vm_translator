# VM Translator - Project 7 & 8 (Nand2Tetris)

This is a VM (Virtual Machine) translator for **Projects 7 & 8** of the [Nand2Tetris course](https://www.nand2tetris.org/).

## Overview

The VM translator converts VM code (`.vm` files) to Hack assembly code (`.asm` files) that can run on the Hack computer platform.

> **Note:** For a simpler implementation that only covers Project 7 functionality, check out the `vm1-project_7` branch.

## Usage

### Basic Usage

```bash
go run main.go -s <source.vm> [-c <compare.asm>]
```

### Examples

```bash
# Convert a single VM file to assembly
go run main.go -s vm1/SimpleAdd.vm

# Process a directory containing multiple .vm files
go run main.go -s vm2/SimpleFunction/

# Compare with expected output
go run main.go -s vm1/StackTest.vm -c vm1/StackTest.cmp
```

### Cleaning Up

```bash
# Clean up generated .asm files
./clean_asm.sh
```

## Supported Commands

The translator now supports:

- **Memory Access Commands**
  - Push/Pop operations for all memory segments (constant, local, argument, this, that, static, temp, pointer)

- **Arithmetic/Logical Commands**
  - Arithmetic: add, sub, neg
  - Logical: eq, gt, lt, and, or, not

- **Program Flow Commands**
  - label - Defines a label
  - goto - Unconditional jump
  - if-goto - Conditional jump

- **Function Calling Commands**
  - function - Function declaration
  - call - Function call
  - return - Return from function

## Project Structure

- `main.go` - Main translator implementation
- `vm1/` - Basic VM code examples (stack operations, arithmetic)
- `vm2/` - Advanced VM code examples (function calls, program flow)
- `clean_asm.sh` - Script to remove generated .asm files

## Nand2Tetris Course

This project is part of the Nand2Tetris course, which teaches how to build a complete computer system from the ground up, starting with NAND gates and ending with a high-level programming language.
