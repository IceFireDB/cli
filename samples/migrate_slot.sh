#!/bin/sh

echo "migrate slot ranges [0,0] to group 2"
../bin/cli -c config.ini slot migrate 0 0 2
echo "done"
