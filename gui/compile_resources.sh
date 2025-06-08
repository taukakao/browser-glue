#!/bin/bash

set -e

mkdir -p "generated"

find . -type f -name "*.blp" | while read -r blp_file; do
    # Get the directory and filename without extension
    output_dir=generated/$(dirname "$blp_file")
    base=$(basename "$blp_file" .blp)

    # Construct the output .ui path
    output_file="$output_dir/$base.ui"

    echo building $output_file
    mkdir -p "$output_dir"
    blueprint-compiler compile --output="$output_file" "$blp_file"
done

echo building generated/resource.gresources
glib-compile-resources --target=generated/resource.gresource gresource.xml --manual-register
