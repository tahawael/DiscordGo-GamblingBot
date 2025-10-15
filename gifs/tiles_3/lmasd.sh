#!/bin/bash

# Navigate to your folder
cd "C:/VisualStudio codes/discord.go/src/gifs/tiles_3" || exit

# Resize all PNG files to 128x128
for file in *.png; do
    magick convert "$file" -resize 128x128 "128_${file}"
done
