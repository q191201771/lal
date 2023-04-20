#!/usr/bin/env bash

# 根据CHANGELOG.md中的最新版本号，决定是否更新t_version.go和以及打git tag
#
# TODO(chef): 新电脑没有gsed导致失败了
#
# 步骤：
# 1. 提交所有代码
# 1-. 检查配置文件中的配置文件版本号和代码中的配置文件版本号是否匹配
# 2. 修改CHANGELOG.md(并手动提交CHANGELOG.md)
# 3. 执行gen_tag.sh

#set -x
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct
export GO111MODULE=on
export GOPROXY=https://goproxy.cn,https://goproxy.io,direct
THIS_FILE=$(readlink -f $0)
THIS_DIR=$(dirname $THIS_FILE)
ROOT_DIR=${THIS_DIR}/..

# CHANGELOG.md中的版本号
NewVersion=`cat ${ROOT_DIR}/CHANGELOG.md| grep '#### v' | head -n 1 | awk  '{print $2}'`
echo 'newest version in CHANGELOG.md: ' $NewVersion

# git tag中的版本号
GitTag=`git tag --sort=version:refname | tail -n 1`
echo "newest version in git tag: " $GitTag

# 源码中的版本号
FileVersion=`cat ${ROOT_DIR}/pkg/base/t_version.go | grep 'const LalVersion' | awk -F\" '{print $2}'`
echo "newest version in t_version.go: " $FileVersion

# CHANGELOG.md和源码中的不一致，更新源码，并提交修改
if [ "$NewVersion" == "$FileVersion" ];then
  echo 'same tag, noop.'
else
  echo 'update t_version.go'
  gsed -i "/^const LalVersion/cconst LalVersion = \"${NewVersion}\"" ${ROOT_DIR}/pkg/base/t_version.go
  git add ${ROOT_DIR}/pkg/base/t_version.go
  git commit -m "${NewVersion} -> t_version.go"
  git push
fi

# CHANGELOG.md和git tag不一致，打新的tag
if [ "$NewVersion" == "$FileVersion" ];then
  echo 'same tag, noop.'
else
  echo 'add tag.' ${NewVersion}
  git tag ${NewVersion}
  git push --tags
fi

