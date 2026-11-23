#!/bin/sh
# Regenerates the README GIFs: ./docs/record.sh  (needs a display + ffmpeg)
set -e
out=docs/media; tmp=$(mktemp -d); mkdir -p $out
rec() { # name, scop args...
  n=$1; shift
  ./scop -width 640 -height 640 -vsync=false -record $tmp/$n -frames 120 "$@" >/dev/null 2>&1
  ffmpeg -loglevel error -y -framerate 30 -i $tmp/$n/f%04d.png \
    -vf "fps=12,scale=320:-1:flags=lanczos,split[a][b];[a]palettegen=max_colors=48:stats_mode=diff[p];[b][p]paletteuse=dither=none:diff_mode=rectangle" $out/$n.gif
}
rec logo_texture -speed 2 models/42.obj
rec teapot_wire -speed 1.5 -wireframe-start models/teapot.obj
rec teapot_uv -speed 1.5 -shade 2 models/teapot.obj
rec teapot_normals -speed 1.5 -shade 1 models/teapot.obj
rec teapot_smooth -speed 1.5 -shade 1 -smooth models/teapot.obj
rec porta_nigra -speed 1.2 models/PortaNigra-from-vertex.obj
rec qilin -speed 1.2 -up z models/qilin.obj
rm -rf $tmp
