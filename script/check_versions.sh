#set -x

# 该脚本用于检查：
#   1. 是否需要更新http api等lalserver的子功能的版本号
#   2. 相应的文档是否需要更新
#
# 包含的模块有：
#   1. 配置
#   2. HTTP-API和HTTP-Notify
#   3. Web-UI
#   4. Go版本
#   5. TODO 依赖版本，目前只有naza

# 已检查的git commit hash id, 或者tag号
# 本地代码会和该版本对比
# 该变量由我手动更新
checked_git_ver="v0.35.41"
#checked_git_ver="5dec8415a6cbe76d0ef230a36f25666da024e368"

# 关注的文件
check_files=(
conf/lalserver.conf.json
conf/lalserver.conf.json.tmpl
pkg/logic/config.go
pkg/rtsp/server.go
pkg/hls/muxer.go

pkg/logic/http_api.go
pkg/logic/http_notify.go
pkg/base/t_http_an__.go
pkg/base/t_http_an__api.go
pkg/base/t_http_an__notify.go

pkg/logic/http_an__lal.html
)

#######################################################################################################################

#set -x
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct
export GO111MODULE=on
export GOPROXY=https://goproxy.cn,https://goproxy.io,direct
THIS_FILE=$(readlink -f $0)
THIS_DIR=$(dirname $THIS_FILE)
ROOT_DIR=${THIS_DIR}/..

compare_with_git_ver=$checked_git_ver

changed_files=`git diff $compare_with_git_ver | grep 'diff --git'`
echo 'changed files: '$changed_files

for(( i=0;i<${#check_files[@]};i++)) 
do
  echo 'checking '${check_files[i]};
  echo $changed_files | grep ${check_files[i]} > /dev/null;
  if [ $? == 0 ]; then
    echo "\033[31m[fuck] "${check_files[i]}" \033[0m";
  else
    echo [ok];
  fi
done;


echo '----------doc conf----------'
cat ${ROOT_DIR}/pkg/base/t_version.go | grep 'ConfVersion ='
cat ${ROOT_DIR}/../lalext/lal_website/ConfigBrief.md| grep 'conf_version' | grep ':'

echo '----------doc http api----------'
cat ${ROOT_DIR}/pkg/base/t_version.go | grep 'HttpApiVersion ='
cat ${ROOT_DIR}/../lalext/lal_website/HTTPAPI.md| grep 'HttpApiVersion' | grep ':'

echo '----------doc http notify----------'
cat ${ROOT_DIR}/pkg/base/t_version.go | grep 'HttpNotifyVersion ='
cat ${ROOT_DIR}/../lalext/lal_website/HTTPNotify.md| grep 'HttpNotifyVersion' | grep ':'

echo '----------doc http web ui----------'
cat ${ROOT_DIR}/pkg/base/t_version.go | grep 'HttpWebUiVersion ='
cat ${ROOT_DIR}/../lalext/lal_website/http_web_ui.md| grep 'HttpWebUiVersion' | grep ':'

echo '----------doc go version----------'
cat ${ROOT_DIR}/go.mod | grep 'go' | grep -v 'module' | grep -v 'require'
cat ${ROOT_DIR}/README.md | grep 'make sure that Go version'
cat ${ROOT_DIR}/../lalext/lal_website/ThirdDeps.md | grep 'Go版本需要'
