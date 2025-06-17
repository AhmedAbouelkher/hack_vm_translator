#!/bin/bash

# Find all .asm files in current directory
asm_files=$(find . -maxdepth 1 -name "*.asm")

if [ -z "$asm_files" ]; then
    echo "No .asm files found in current directory"
    exit 0
fi

echo "Found .asm files to remove:"
echo "$asm_files"
echo ""

# Ask for confirmation
read -p "Are you sure you want to remove these files? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Operation cancelled"
    exit 0
fi

# Remove each .asm file
for file in $asm_files; do
    echo "Removing: $file"
    rm "$file"
    if [ $? -eq 0 ]; then
        echo "✓ Successfully removed $file"
    else
        echo "✗ Failed to remove $file"
    fi
done

echo ""
echo "Cleanup completed" 