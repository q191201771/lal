#!/usr/bin/env bash

make image
docker run -it -p 8084:8084 -p 8080:8080 -p 1935:1935 -p 5544:5544 -p 8083:8083 lal /lal/bin/lalserver
