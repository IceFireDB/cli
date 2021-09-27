#!/bin/sh
echo "slots initializing..."
../bin/cli -c config.ini slot init -f true
echo "done"

echo "set slot ranges to server groups..."
../bin/cli -c config.ini slot range-set 0 63 1 online
../bin/cli -c config.ini slot range-set 64 127 2 online
echo "done"

