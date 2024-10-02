#!/bin/bash

# Function to generate a clean, lowercase filename
clean_filename() {
    echo "$1" | \
    tr '[:upper:]' '[:lower:]' | \
    sed -e 's/[^[:alnum:][:space:]-]/ /g' | \
    sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' | \
    sed -e 's/[[:space:]]\+/-/g'
}

# Loop through all SVG files in the current directory
for svg_file in *.svg *.SVG; do
    # Check if the file exists (in case no SVG files are found)
    [ -e "$svg_file" ] || continue
    
    # Generate the new SVG filename
    new_svg_name=$(clean_filename "${svg_file%.svg}").svg
    
    # Rename the SVG file if the new name is different
    if [ "$svg_file" != "$new_svg_name" ]; then
        mv "$svg_file" "$new_svg_name"
        echo "Renamed $svg_file to $new_svg_name"
    fi
    
    # Generate the PNG filename
    png_file=$(clean_filename "${svg_file%.svg}").png
    
    # Convert SVG to PNG using Inkscape
    inkscape --export-type=png \
             --export-filename="$png_file" \
             --export-width=3600 \
             "$new_svg_name"
    
    echo "Converted $new_svg_name to $png_file"
done

echo "Renaming and conversion complete!"
