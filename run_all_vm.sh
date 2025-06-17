#!/bin/bash

# Find all .vm files in current directory, sort them alphabetically
vm_files=$(find . -maxdepth 1 -name "*.vm" | sort)

if [ -z "$vm_files" ]; then
    echo "No .vm files found in current directory"
    exit 1
fi

echo "Found .vm files:"
echo "$vm_files"
echo ""

# Process each .vm file
for file in $vm_files; do
    echo "Processing: $file"
    go run main.go -s "$file"
    if [ $? -eq 0 ]; then
        echo "✓ Successfully processed $file"
    else
        echo "✗ Failed to process $file"
    fi
    echo ""
done

echo "All files processed" 