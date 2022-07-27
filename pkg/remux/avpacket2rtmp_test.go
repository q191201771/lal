// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux_test

import (
	"encoding/hex"
	"testing"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
)

// #85
func TestCase1(t *testing.T) {
	ps := []string{
		// sps
		"0000002c67640032ad84010c20086100430802184010c200843b5014005ad370101014000003000400000300ca100002",
		// pps
		"0000000468ee3cb0",
	}
	golden := []base.AvPacket{
		{
			Timestamp:   10340642,
			PayloadType: base.AvPacketPtAvc,
		},
		{
			Timestamp:   10340642,
			PayloadType: base.AvPacketPtAvc,
		},
	}
	for i := range ps {
		p, _ := hex.DecodeString(ps[i])
		golden[i].Payload = p
	}

	remuxer := remux.NewAvPacket2RtmpRemuxer().WithOnRtmpMsg(func(msg base.RtmpMsg) {
		remux.Log.Debugf("%+v", msg)
	})
	for _, p := range golden {
		remuxer.FeedAvPacket(p)
	}
}

func TestCase2(t *testing.T) {
	ps := []string{
		// vps
		"0000001840010c01ffff016000000300b0000003000003007bac0901",
		// sps
		"00000024420101016000000300b0000003000003007ba003c08010e58dae4914bf37010101008001",
		// pps
		"0000000c4401c0f2c68d03b240000003",
		// 非关键帧
		"0000000c4e01e504ebc3000080000003",
	}
	golden := []base.AvPacket{
		{
			Timestamp:   25753900,
			PayloadType: base.AvPacketPtHevc,
		},
		{
			Timestamp:   25753900,
			PayloadType: base.AvPacketPtHevc,
		},
		{
			Timestamp:   25753900,
			PayloadType: base.AvPacketPtHevc,
		},
		{
			Timestamp:   25753900,
			PayloadType: base.AvPacketPtHevc,
		},
	}

	for i := range ps {
		p, _ := hex.DecodeString(ps[i])
		golden[i].Payload = p
	}

	remuxer := remux.NewAvPacket2RtmpRemuxer().WithOnRtmpMsg(func(msg base.RtmpMsg) {
		remux.Log.Debugf("%+v", msg)
	})
	for _, p := range golden {
		remuxer.FeedAvPacket(p)
	}
}
