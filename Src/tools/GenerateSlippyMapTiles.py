# -*- coding: utf-8 -*-

import argparse, magic, re, os, math, glob, shutil

'''
GENERATE SLIPPY MAP TILES
Jeff Thompson | 2016 | jeffreythompson.org

Takes a large image as the input, outputs map tiles
at the appropriate size and file structure for use
in frameworks like leaflet.js, MapBox, etc.

ARGS:
input_file		large image file to split (JPG, PNG, or TIFF)
zoom_level          	zoom level(s) to generate (0 to 18); either
			integer or range (ex: 2-6)
output_folder       	folder name to write tiles to (will be created
			if does not exist)

OPTIONAL:
-h, --help          	show this help message and exit
-w --resize_width	dimension in pixels for outputted tiles (default 256px)
-q, --quiet         	suppress all output from program (useful for
			integrating into larger projects)

DETAILS:
Resulting tiles are 256px square, regardless of the
size of the source image. The number of tiles wide/
high is determined by the "zoom level", which is
2^zoom. In other words, a zoom level of 3 = 8 tiles,
each resized to 256 pixels square.

Way more info here:
http://wiki.openstreetmap.org/wiki/Slippy_map_
tilenames#Resolution_and_Scale

REQUIRES:
ImageMagick and Python bindings for splitting
images, resizing tiles, etc
http://www.imagemagick.org
https://github.com/ahupp/python-magic

FILE STRUCTURE
Slippy maps require tiles to be stored in a specific
file structure:
	output_folder/zoom_level/x/y.png

This is the standard arrangement (some frameworks let
you specify others), and should be noted in your Javascript.
For example, if using leaflet.js, you would use:
	tiles/{z}/{x}/{y}.png

ADDING MORE ZOOM LEVELS
Want to add more levels? Just run this script again; it
will append the new zoom level to the same location.

CREATING A SOURCE IMAGE
If combining many smaller images, the easiest method
is to use ImageMagick's 'montage' command.

Your images should be the same size, or at least the
same height. You can do this using ImageMagick as well:
	mogrify -geometry x400 *.jpg

Arguments:
	x400			height to set images to
	*.jpg 			gets all jpg images from a folder

Then combine into a single image:
	montage *.jpg -gravity center -tile NxN -geometry +0+0 output.jpg

Arguments:
	*.jpg 			gets all jpg images from a folder
	-gravity		centers rows/columns
	-tile			how many images per row/column in final image
	-geometry		no extra space between images (or +N+N for padding)
	-background		none or "rgb(255,255,255)"
	output.jpg		output filename and format

VERY LARGE IMAGES:
When working with extra big images, ImageMagick makes
some suggestions where RAM may run out:
http://www.imagemagick.org/Usage/files/#massive

'''

# ==============

def power_of(num, base):
	''' checks if a number is a power another '''
	while(num % base == 0):
		num = num / base
	return num == 1

def generate(input_file, output_folder, zoom_level, resize_width, quiet):
	''' generates slippy map tiles from large image '''

	# how many tiles will that be?
	num_tiles = pow(2, zoom_level)
	if not quiet: print 'Zoom level ' + str(zoom_level) + ' = ' + str(num_tiles) + ' tiles'

	# get image dims (without loading into memory)
	# via: http://stackoverflow.com/a/19035508/1167783
	if not quiet: print 'Getting source image dimensions...'
	t = magic.from_file(input_file)
	try:
		if input_file.endswith('.jpg') or input_file.endswith('.jpeg'):
			dims = re.search(', (\d+)x(\d+)', t)
			width = int(dims.group(1))
			height = int(dims.group(2))
		elif input_file.endswith('.tif') or input_file.endswith('.tiff'):
			width = int(re.search('width=(\d+)', t).group(1))
			height = int(re.search('height=(\d+),', t).group(1))
		elif input_file.endswith('.png'):
			dims = re.search(', (\d+) x (\d+)', t)
			width = int(dims.group(1))
			height = int(dims.group(2))
		else:
			if not quiet: print 'ERROR: Unknown source image type; JPG, TIFF, or PNG only! Quitting...'
			exit(0)
	except:
		if not quiet: print 'ERROR: Could not parse source image dims! Quitting...'
		exit(0)

	if not quiet: print '- ' + str(width) + ' x ' + str(height) + ' pixels'

	# errors and warnings
	if not power_of(int(width), 2):
		if not quiet: print 'WARNING: Source image dims should be power of 2! Continuing anyway...'
	if width != height:
		if not quiet: print 'ERROR: Source image should be square! Quitting...'
		exit(0)

	# get details for ImageMagick
	if not quiet: print 'Splitting to...'
	tile_width = int(math.ceil(width / num_tiles))
	if not quiet: print '- ' + str(tile_width) + ' x ' + str(tile_width) + ' px tiles'
	pad = len(str(num_tiles * num_tiles))

	# split using ImageMagic, then resize to the expected tile size
	# use filename padding so glob gets them in the right order
	if not os.path.exists(output_folder):
		os.mkdir(output_folder)
	cmd = 'magick convert ' + input_file + ' -quiet -crop ' + str(tile_width) + 'x' + str(tile_width) + ' -resize ' + str(resize_width) + 'x' + str(resize_width) + ' ' + output_folder + '/%0' + str(pad) + 'd.png'
	os.popen(cmd)
	if not quiet: print '- done!'

	# rename/move images into tile server format
	if not quiet: print 'Moving files into column folders...'

	# 1. make cols
	for x in range(0, num_tiles):
		folder = output_folder + '/' + str(zoom_level) + '/' + str(x)
		if not os.path.exists(folder):
			os.makedirs(folder)

	# 2. move tiles into their column folders
	tiles = glob.glob(output_folder + '/*.png')
	tiles = sorted(tiles)  # tjr: sort first because glob has arbitrary ordering
	for i, tile in enumerate(tiles):
		col = i % num_tiles
		f = i / num_tiles
		dst = output_folder + '/' + str(zoom_level) + '/' + str(col) + '/' + str(f) + '.png'
		# print "i = ", i
		# print "tile = ", tile
		# print "move ", tile, dst
		# print ""
		shutil.move(tile, dst)
	if not quiet: print '- done!'

# ==============

if __name__ == '__main__':
	p = argparse.ArgumentParser(description='Takes a large image as the input, outputs map tiles at the appropriate size and file structure for use in frameworks like leaflet.js, MapBox, etc. Much more info in the source code.', usage='python GenerateSlippyMapTiles.py input_file zoom_level output_folder [options]')
	p.add_argument('input_file', help='large image file to split (JPG, PNG, or TIFF)')
	p.add_argument('zoom_level', help='zoom level(s) to generate (0 to 18); either integer or range (ex: 2-6)')
	p.add_argument('output_folder', help='folder name to write tiles to (will be created if does not exist)')
	p.add_argument('-w', '--resize_width', help='dimension in pixels for outputted tiles (default 256px)', metavar='', type=int, default=256)
	p.add_argument('-q', '--quiet', help='suppress all output from program (useful for integrating into larger projects)', action='store_true')
	args = p.parse_args()

	input_file = args.input_file
	zoom_level = args.zoom_level
	output_folder = args.output_folder
	resize_width = args.resize_width
	quiet = args.quiet

	if not quiet: print 'GENERATING SLIPPY-MAP TILES'
	if not quiet: print ('- ' * 14)

	# if multiple zoom levels, run them all
	# otherwise, run just once
	if '-' in zoom_level:
		try:
			match = re.search(r'([0-9]+)-([0-9]+)', zoom_level)
			zoom_min = int(match.group(1))
			zoom_max = int(match.group(2))
		except:
			if not quiet: print "ERROR: Couldn't parse zoom levels; should be int or 'min-max'! Quitting..."
			exit(0)
		for z in range(zoom_min, zoom_max+1):
			generate(input_file, output_folder, z, resize_width, quiet)
			if not quiet: print ('- ' * 14)
	else:
		generate(input_file, output_folder, int(zoom_level), resize_width, quiet)
		if not quiet: print ('- ' * 14)

	# that's it!
	if not quiet: print 'FINISHED!'

