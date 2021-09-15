#!/bin/sh
../bin/cli -c config.ini -L ./log/cconfig.log server add 1 127.0.0.1:6398
../bin/cli -c config.ini -L ./log/cconfig.log server add 2 127.0.0.1:6399

