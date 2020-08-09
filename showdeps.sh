#!/usr/bin/env bash

# lalserver pushrtmp pullrtmp pullhttpflv innertest
# logic
# rtsp
# hls httpflv rtmp rtprtcp sdp
# base aac avc hevc

for d in $(go list ./pkg/...); do
  echo "-----"$d"-----"
  # 只看依赖lal自身的哪些package
  # package依赖自身这个package的过滤掉
  # 依赖pkg/base这个基础package的过滤掉
  #go list -deps $d | grep 'q191201771/lal' | grep -v $d | grep -v 'q191201771/lal/pkg/base'
  #go list -deps $d | grep 'q191201771/lal' | grep -v $d
  go list -deps $d | grep 'q191201771/naza' | grep -v $d
done

for d in $(go list ./app/...); do
  echo "-----"$d"-----"
  #go list -deps $d | grep 'q191201771/lal' | grep -v $d
  go list -deps $d | grep 'q191201771/naza' | grep -v $d
done
