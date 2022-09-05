package logic

import (
	"github.com/q191201771/lal/pkg/base"
)

type IAuthentication interface {
	OnPubStart(info base.PubStartInfo) error
	OnSubStart(info base.SubStartInfo) error
	OnHls(streamName, urlParam string) error
}