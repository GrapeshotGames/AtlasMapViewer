#!/bin/bash
#
# Uses ImageMagick to replace black with other colors for a given image.
# Example:
#   $ mkdir bed && ./addcolors.bash bed.png bed/

src=$1
out=$2

for color in red green yellow blue orange purple cyan magenta lime \
    pink teal lavender brown beige maroon olive coral navy
do
    convert $src -fill $color -opaque black $out$color.png
done
