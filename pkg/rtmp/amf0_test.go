// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/q191201771/lal/pkg/base"

	. "github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/fake"
)

func TestAmf0_WriteNumber_ReadNumber(t *testing.T) {
	cases := []float64{
		0,
		1,
		0xff,
		1.2,
	}

	for _, item := range cases {
		out := &bytes.Buffer{}
		err := Amf0.WriteNumber(out, item)
		assert.Equal(t, nil, err)
		v, l, err := Amf0.ReadNumber(out.Bytes())
		assert.Equal(t, item, v)
		assert.Equal(t, l, 9)
		assert.Equal(t, nil, err)
	}
}

func TestAmf0_WriteString_ReadString(t *testing.T) {
	cases := []string{
		"a",
		"ab",
		"111",
		"~!@#$%^&*()_+",
	}
	for _, item := range cases {
		out := &bytes.Buffer{}
		err := Amf0.WriteString(out, item)
		assert.Equal(t, nil, err)
		v, l, err := Amf0.ReadString(out.Bytes())
		assert.Equal(t, item, v)
		assert.Equal(t, l, len(item)+3)
		assert.Equal(t, nil, err)
	}

	longStr := strings.Repeat("1", 65536)
	out := &bytes.Buffer{}
	err := Amf0.WriteString(out, longStr)
	assert.Equal(t, nil, err)
	v, l, err := Amf0.ReadString(out.Bytes())
	assert.Equal(t, longStr, v)
	assert.Equal(t, l, len(longStr)+5)
	assert.Equal(t, nil, err)
}

func TestAmf0_WriteObject_ReadObject(t *testing.T) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{Key: "air", Value: 3},
		{Key: "ban", Value: "cat"},
		{Key: "dog", Value: true},
	}
	err := Amf0.WriteObject(out, objs)
	assert.Equal(t, nil, err)
	v, _, err := Amf0.ReadObject(out.Bytes())
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(v))
	assert.Equal(t, float64(3), v.Find("air"))
	assert.Equal(t, "cat", v.Find("ban"))
	assert.Equal(t, true, v.Find("dog"))
}

func TestAmf0_WriteNull_readNull(t *testing.T) {
	out := &bytes.Buffer{}
	err := Amf0.WriteNull(out)
	assert.Equal(t, nil, err)
	l, err := Amf0.ReadNull(out.Bytes())
	assert.Equal(t, 1, l)
	assert.Equal(t, nil, err)
}

func TestAmf0_WriteBoolean_ReadBoolean(t *testing.T) {
	cases := []bool{true, false}

	for i := range cases {
		out := &bytes.Buffer{}
		err := Amf0.WriteBoolean(out, cases[i])
		assert.Equal(t, nil, err)
		v, l, err := Amf0.ReadBoolean(out.Bytes())
		assert.Equal(t, cases[i], v)
		assert.Equal(t, 2, l)
		assert.Equal(t, nil, err)
	}
}

func TestAmf0_ReadArray(t *testing.T) {
	gold := []byte{0x08, 0x00, 0x00, 0x00, 0x10, 0x00, 0x08, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x77, 0x69, 0x64, 0x74, 0x68, 0x00, 0x40, 0x88, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x00, 0x40, 0x74, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0d, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x64, 0x61, 0x74, 0x61, 0x72, 0x61, 0x74, 0x65, 0x00, 0x40, 0x69, 0xe8, 0x50, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x66, 0x72, 0x61, 0x6d, 0x65, 0x72, 0x61, 0x74, 0x65, 0x00, 0x40, 0x39, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0c, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x63, 0x6f, 0x64, 0x65, 0x63, 0x69, 0x64, 0x00, 0x40, 0x1c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0d, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x64, 0x61, 0x74, 0x61, 0x72, 0x61, 0x74, 0x65, 0x00, 0x40, 0x3d, 0x54, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x72, 0x61, 0x74, 0x65, 0x00, 0x40, 0xe5, 0x88, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x69, 0x7a, 0x65, 0x00, 0x40, 0x30, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0x73, 0x74, 0x65, 0x72, 0x65, 0x6f, 0x01, 0x01, 0x00, 0x0c, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x63, 0x6f, 0x64, 0x65, 0x63, 0x69, 0x64, 0x00, 0x40, 0x24, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b, 0x6d, 0x61, 0x6a, 0x6f, 0x72, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x64, 0x02, 0x00, 0x04, 0x69, 0x73, 0x6f, 0x6d, 0x00, 0x0d, 0x6d, 0x69, 0x6e, 0x6f, 0x72, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x02, 0x00, 0x03, 0x35, 0x31, 0x32, 0x00, 0x11, 0x63, 0x6f, 0x6d, 0x70, 0x61, 0x74, 0x69, 0x62, 0x6c, 0x65, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x64, 0x73, 0x02, 0x00, 0x10, 0x69, 0x73, 0x6f, 0x6d, 0x69, 0x73, 0x6f, 0x32, 0x61, 0x76, 0x63, 0x31, 0x6d, 0x70, 0x34, 0x31, 0x00, 0x07, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x65, 0x72, 0x02, 0x00, 0x0d, 0x4c, 0x61, 0x76, 0x66, 0x35, 0x37, 0x2e, 0x38, 0x33, 0x2e, 0x31, 0x30, 0x30, 0x00, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x69, 0x7a, 0x65, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09}

	ops, l, err := Amf0.ReadArray(gold)
	assert.Equal(t, nil, err)
	assert.Equal(t, 16, len(ops))

	var fn int
	var fe error
	var fi interface{}
	fn, fe = ops.FindNumber("duration")
	assert.Equal(t, 0, fn)
	assert.Equal(t, nil, fe)
	fn, fe = ops.FindNumber("width")
	assert.Equal(t, 768, fn)
	assert.Equal(t, nil, fe)
	fn, fe = ops.FindNumber("height")
	assert.Equal(t, 320, fn)
	assert.Equal(t, nil, fe)
	fi = ops.Find("videodatarate")
	assert.Equal(t, true, fi.(float64) > 207 && fi.(float64) < 208)
	fn, fe = ops.FindNumber("framerate")
	assert.Equal(t, 25, fn)
	assert.Equal(t, nil, fe)
	fn, fe = ops.FindNumber("videocodecid")
	assert.Equal(t, 7, fn)
	assert.Equal(t, nil, fe)
	fi = ops.Find("audiodatarate")
	assert.Equal(t, true, fi.(float64) > 29 && fi.(float64) < 30)
	assert.Equal(t, nil, fe)
	// skip 7, 8
	fi = ops.Find("stereo")
	assert.Equal(t, true, fi.(bool))
	// skip 10
	fi = ops.Find("major_brand")
	assert.Equal(t, "isom", fi.(string))
	fi = ops.Find("minor_version")
	assert.Equal(t, "512", fi.(string))
	// skip 13, 14
	fn, fe = ops.FindNumber("filesize")
	assert.Equal(t, 0, fn)
	assert.Equal(t, nil, fe)

	assert.Equal(t, 359, len(gold))
	assert.Equal(t, 359, l)

	h := hex.EncodeToString([]byte(ops.DebugString()))
	assert.Equal(t, "6475726174696f6e3a20300a77696474683a203736380a6865696768743a203332300a766964656f64617461726174653a203230372e3235393736353632350a6672616d65726174653a2032350a766964656f636f64656369643a20370a617564696f64617461726174653a2032392e333239313031353632350a617564696f73616d706c65726174653a2034343130300a617564696f73616d706c6573697a653a2031360a73746572656f3a20747275650a617564696f636f64656369643a2031300a6d616a6f725f6272616e643a2069736f6d0a6d696e6f725f76657273696f6e3a203531320a636f6d70617469626c655f6272616e64733a2069736f6d69736f32617663316d7034310a656e636f6465723a204c61766635372e38332e3130300a66696c6573697a653a20300a", h)
	Log.Debug(ops)
}

func TestAmf0_ReadStrictArray(t *testing.T) {
	goldIn := "0a000000870040a80c00000000000040a8b6000000000000413218e2000000000041426d0080000000004149257b8000000000414e64f900000000004152521c0000000000415716838000000000415acb29c000000000415e1be40000000000415ea7070000000000416113e460000000004162e46da00000000041638a54c00000000041642de1a0000000004165cac380000000004167b1eb0000000000416991006000000000416b469d4000000000416d3b82c000000000416f2b11e000000000417091b7b00000000041715ec1000000000041725faf2000000000417363bf900000000041743eb9a0000000004174d44190000000004175d45c00000000004176d5a0a0000000004177e0cdd0000000004178e375d00000000041791f395000000000417986b0e000000000417aa6f70000000000417bb76f0000000000417cc5bdf000000000417dcc6fd000000000417eba306000000000417ee78bb000000000417fdc14e00000000041806fea10000000004180ec76b000000000418171ef70000000004181f387d800000000418256dc40000000004182d91b70000000004182d91b700000000041835b42d8000000004183d05868000000004184491b30000000004184bd2e580000000041851b566800000000418531ec100000000041855a0610000000004185dbe0880000000041865e2b4800000000418693cd78000000004186d2337800000000418759be38000000004187d235680000000041884a7940000000004188c4376000000000418939ea20000000004189b3be4000000000418a27777000000000418a6e12f000000000418a8d840800000000418ad528f000000000418b1720f000000000418b8e449000000000418c105a9000000000418c32d32800000000418c7f8d4000000000418d007ff000000000418d5ec58000000000418da38ee000000000418e1991d800000000418e8d9af800000000418ea34b9800000000418f15a97000000000418f8adaa800000000418fcd60f800000000418ff5da8800000000419002daf800000000419002daf8000000004190434fa000000000419085a5b0000000004190c016b4000000004190fbfc180000000041911fc314000000004191360de00000000041914a9ee800000000419189f434000000004191cc161c00000000419206f2cc000000004192462a90000000004192568e20000000004192966560000000004192d732e8000000004192ec1b600000000041931197180000000041934de648000000004193879dc40000000041939fffd4000000004193cdb5c800000000419404836c000000004194291060000000004194383830000000004194776d20000000004194885538000000004194c9519000000000419503b5580000000041952e1b680000000041955202000000000041959498b4000000004195d1c5f8000000004196115c000000000041962e921400000000419643b3bc00000000419654649c0000000041968b006c000000004196be3bd8000000004196eacbc8000000004197182d54000000004197546634000000004197968854000000004197d546380000000041980dfa2000000000419851517c0000000041985ee4f000000000419881237c000000004198bd64f0000000004198fa77e4000000004199376594000000004199538970000000000574696d65730a0000008700000000000000000000000000000000000000401c7ae147ae147b0040311eb851eb851f00403b1eb851eb851f004041c7ae147ae1480040468a3d70a3d70a00404b8a3d70a3d70a004050451eb851eb850040523851eb851eb8004052b851eb851eb80040553851eb851eb8004057b851eb851eb80040588ccccccccccd00405919999999999a00405b99999999999a00405e19999999999a0040604ccccccccccd0040618ccccccccccd004062cccccccccccd0040640ccccccccccd00406548f5c28f5c29004065eccccccccccd0040672ccccccccccd0040686ccccccccccd0040697c28f5c28f5c004069f1eb851eb85200406b31eb851eb85200406c71eb851eb85200406db1eb851eb85200406ef1eb851eb85200406f347ae147ae1400406f800000000000004070600000000000004071000000000000004071a00000000000004072400000000000004072d51eb851eb85004072f0a3d70a3d7100407390a3d70a3d7100407430a3d70a3d71004074d0a3d70a3d7100407570a3d70a3d7100407610a3d70a3d710040768bdf3b645a1d0040772b3b645a1cac0040772b3b645a1cac004077cb3b645a1cac0040786b3b645a1cac0040790b3b645a1cac004079ab3b645a1cac00407a2fb645a1cac100407a4f126e978d5000407a738d4fdf3b6400407b138d4fdf3b6400407bb38d4fdf3b6400407bf6c083126e9800407c28083126e97900407cc8083126e97900407d68083126e97900407e08083126e97900407ea8083126e97900407f48083126e97900407fe8083126e9790040804404189374bc00408063604189374c00408074f9db22d0e500408093604189374c004080ba6a7ef9db230040810a6a7ef9db230040815a6a7ef9db23004081707ef9db22d100408189c6a7ef9db2004081d9c6a7ef9db2004082182d0e56041900408233b22d0e560400408283b22d0e5604004082d3b22d0e5604004082e0d0e56041890040830c56041893750040835c560418937500408384f9db22d0e500408392bc6a7ef9db0040839a6a7ef9db230040839a6a7ef9db23004083ea6a7ef9db230040843a6a7ef9db230040848a6a7ef9db23004084da6a7ef9db230040850a6a7ef9db230040851bb22d0e56040040852e9374bc6a7f0040857e9374bc6a7f004085ce9374bc6a7f0040861e9374bc6a7f0040866e9374bc6a7f00408688d70a3d70a4004086d8d70a3d70a400408728d70a3d70a40040874128f5c28f5c00408760851eb851ec004087b0851eb851ec00408800851eb851ec0040881f3d70a3d70a0040883f8f5c28f5c30040888f8f5c28f5c3004088c6999999999a004088d9cccccccccd00408929cccccccccd004089383333333333004089883333333333004089d8333333333300408a0d51eb851eb800408a2d51eb851eb800408a7d51eb851eb800408acd51eb851eb800408b1d51eb851eb800408b43b851eb851f00408b51cccccccccd00408b63666666666600408bb3666666666600408c03666666666600408c42c28f5c28f600408c68333333333300408cb8333333333300408d08333333333300408d58333333333300408da8333333333300408df8333333333300408e0270a3d70a3d00408e1e47ae147ae100408e6e47ae147ae100408ebe47ae147ae100408f017ae147ae1400408f1f3d70a3d70a000009000009"
	b, err := hex.DecodeString(goldIn)
	assert.Equal(t, nil, err)

	ops, l, err := Amf0.ReadStrictArray(b)
	assert.Equal(t, 135, len(ops))
	goldOut := hex.EncodeToString([]byte(ops.DebugString()))
	assert.Equal(t, "3a20333037380a3a20333136330a3a20312e313836303138652b30360a3a20322e343135313035652b30360a3a20332e323935393931652b30360a3a20332e393833383538652b30360a3a20342e383032363732652b30360a3a20362e303532333636652b30360a3a20372e303233373833652b30360a3a20372e3839323838652b30360a3a20382e303335333536652b30360a3a20382e393533363335652b30360a3a20392e393035303035652b30360a3a20312e30323434373734652b30370a3a20312e30353739373235652b30370a3a20312e31343235333038652b30370a3a20312e32343233652b30370a3a20312e33343034313633652b30370a3a20312e34333030333934652b30370a3a20312e353332363233652b30370a3a20312e36333431313335652b30370a3a20312e37333734303735652b30370a3a20312e38323133393034652b30370a3a20312e393236363239652b30370a3a20322e30333331353133652b30370a3a20322e31323238343432652b30370a3a20322e31383430393231652b30370a3a20322e323838393932652b30370a3a20322e333934333639652b30370a3a20322e35303338303435652b30370a3a20322e36303937353031652b30370a3a20322e36333432323933652b30370a3a20322e36373636303934652b30370a3a20322e37393436383634652b30370a3a20322e39303632383936652b30370a3a20332e30313730303739652b30370a3a20332e31323436303737652b30370a3a20332e323231393931652b30370a3a20332e32343035363931652b30370a3a20332e333430373331652b30370a3a20332e34343731323334652b30370a3a20332e35343931353432652b30370a3a20332e36353834393432652b30370a3a20332e37363436353837652b30370a3a20332e38343630323936652b30370a3a20332e39353237323738652b30370a3a20332e39353237323738652b30370a3a20342e30353933343939652b30370a3a20342e31353532363533652b30370a3a20342e32353431393236652b30370a3a20342e33343932383131652b30370a3a20342e34323634313431652b30370a3a20342e34343439313534652b30370a3a20342e34373737363636652b30370a3a20342e35383431343235652b30370a3a20342e36393038373737652b30370a3a20342e37333438313433652b30370a3a20342e37383539333131652b30370a3a20342e38393639363731652b30370a3a20342e39393536353235652b30370a3a20352e30393431373336652b30370a3a20352e31393339303532652b30370a3a20352e32393033323336652b30370a3a20352e33393031323536652b30370a3a20352e34383439323632652b30370a3a20352e35343237363738652b30370a3a20352e35363835323439652b30370a3a20352e36323732313538652b30370a3a20352e36383132353734652b30370a3a20352e37373838353632652b30370a3a20352e38383534323236652b30370a3a20352e39313336363133652b30370a3a20352e393736353136652b30370a3a20362e30383231353032652b30370a3a20362e31353933373736652b30370a3a20362e32313537323736652b30370a3a20362e33313234303237652b30370a3a20362e34303734353931652b30370a3a20362e34323532323735652b30370a3a20362e35313839313636652b30370a3a20362e36313439323035652b30370a3a20362e36363934313735652b30370a3a20362e37303235373435652b30370a3a20362e37313535363436652b30370a3a20362e37313535363436652b30370a3a20362e38323131363838652b30370a3a20362e393239383534652b30370a3a20372e30323536303435652b30370a3a20372e31323337333832652b30370a3a20372e31383233353537652b30370a3a20372e32313838373932652b30370a3a20372e32353235373534652b30370a3a20372e33353633343035652b30370a3a20372e34363436393139652b30370a3a20372e35363131333135652b30370a3a20372e36363437303736652b30370a3a20372e36393135353932652b30370a3a20372e373936313536652b30370a3a20372e393032333239652b30370a3a20372e39333635383438652b30370a3a20372e39393739393734652b30370a3a20382e30393638303832652b30370a3a20382e31393133373133652b30370a3a20382e32333133323035652b30370a3a20382e333036323133652b30370a3a20382e33393630303237652b30370a3a20382e34353538383732652b30370a3a20382e343830373138652b30370a3a20382e353834323736652b30370a3a20382e36313139373538652b30370a3a20382e37313834343834652b30370a3a20382e38313431313432652b30370a3a20382e38383335383032652b30370a3a20382e39343234652b30370a3a20392e30353134393839652b30370a3a20392e313531373331652b30370a3a20392e32353539313034652b30370a3a20392e33303337373031652b30370a3a20392e33333833393139652b30370a3a20392e33363537333833652b30370a3a20392e34353532303931652b30370a3a20392e35333931343738652b30370a3a20392e36313231353836652b30370a3a20392e36383635313039652b30370a3a20392e37383531373839652b30370a3a20392e38393335333137652b30370a3a20392e39393633323738652b30370a3a20312e3030383932323936652b30380a3a20312e3031393935363135652b30380a3a20312e3032323138303434652b30380a3a20312e3032373739313033652b30380a3a20312e3033373636333332652b30380a3a20312e3034373636393639652b30380a3a20312e3035373635323231652b30380a3a20312e3036323236323638652b30380a", goldOut)
	assert.Equal(t, 1220, l)
	assert.Equal(t, nil, err)
}

func TestAmf0_ReadCase1(t *testing.T) {
	// ZLMediaKit connect result的object中存在null type
	// https://github.com/q191201771/lal/issues/102
	//
	gold := "030000000000b614000000000200075f726573756c74003ff000000000000003000c6361706162696c697469657300403f0000000000000006666d7356657202000d464d532f332c302c312c313233000009030004636f646502001d4e6574436f6e6e656374696f6e2e436f6e6e6563742e53756363657373000b6465736372697074696f6e020015436f6e6e656374696f6e207375636365656465642e00056c6576656c020006737461747573000e6f626a656374456e636f64696e6705000009"
	goldbytes, err := hex.DecodeString(gold)
	assert.Equal(t, nil, err)
	index := 12

	s, l, err := Amf0.ReadString(goldbytes[index:])
	assert.Equal(t, nil, err)
	assert.Equal(t, "_result", s)
	index += l

	n, l, err := Amf0.ReadNumber(goldbytes[index:])
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(1), n)
	index += l

	o, l, err := Amf0.ReadObject(goldbytes[index:])
	assert.Equal(t, nil, err)
	i, err := o.FindNumber("capabilities")
	assert.Equal(t, nil, err)
	assert.Equal(t, 31, i)
	s, err = o.FindString("fmsVer")
	assert.Equal(t, nil, err)
	assert.Equal(t, "FMS/3,0,1,123", s)
	index += l

	o, l, err = Amf0.ReadObject(goldbytes[index:])
	assert.Equal(t, nil, err)
	s, err = o.FindString("code")
	assert.Equal(t, nil, err)
	assert.Equal(t, "NetConnection.Connect.Success", s)
	s, err = o.FindString("description")
	assert.Equal(t, nil, err)
	assert.Equal(t, "Connection succeeded.", s)
	s, err = o.FindString("level")
	assert.Equal(t, nil, err)
	assert.Equal(t, "status", s)
	index += l

	assert.Equal(t, len(goldbytes), index)
}

func TestAmf0Corner(t *testing.T) {
	var (
		mw   *fake.Writer
		err  error
		b    []byte
		str  string
		l    int
		num  float64
		flag bool
		objs []ObjectPair
	)

	mw = fake.NewWriter(fake.WriterTypeReturnError)
	err = Amf0.WriteNumber(mw, 0)
	assert.IsNotNil(t, err)

	mw = fake.NewWriter(fake.WriterTypeReturnError)
	err = Amf0.WriteBoolean(mw, true)
	assert.IsNotNil(t, err)

	// WriteString 调用 三次写
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{0: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, "0")
	assert.IsNotNil(t, err)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{1: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, "1")
	assert.IsNotNil(t, err)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{2: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, "2")
	assert.IsNotNil(t, err)
	longStr := strings.Repeat("1", 65536)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{0: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{1: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{2: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)

	objs = []ObjectPair{
		{Key: "air", Value: 3},
		{Key: "ban", Value: "cat"},
		{Key: "dog", Value: true},
	}
	for i := uint32(0); i < 14; i++ {
		mw = fake.NewWriter(fake.WriterTypeDoNothing)
		mw.SetSpecificType(map[uint32]fake.WriterType{i: fake.WriterTypeReturnError})
		err = Amf0.WriteObject(mw, objs)
		assert.IsNotNil(t, err)
	}

	b = nil
	str, l, err = Amf0.ReadStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))
	b = []byte{1, 1}
	str, l, err = Amf0.ReadStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))

	b = nil
	str, l, err = Amf0.ReadLongStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))
	b = []byte{1, 1, 1, 1}
	str, l, err = Amf0.ReadLongStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))

	b = nil
	str, l, err = Amf0.ReadString(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))
	b = []byte{1}
	str, l, err = Amf0.ReadString(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfInvalidType))

	b = nil
	num, l, err = Amf0.ReadNumber(b)
	assert.Equal(t, int(num), 0)
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))
	str = strings.Repeat("1", 16)
	b = []byte(str)
	num, l, err = Amf0.ReadNumber(b)
	assert.Equal(t, int(num), 0)
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfInvalidType))

	b = nil
	flag, l, err = Amf0.ReadBoolean(b)
	assert.Equal(t, flag, false)
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))
	b = []byte{0, 0}
	flag, l, err = Amf0.ReadBoolean(b)
	assert.Equal(t, flag, false)
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfInvalidType))

	b = nil
	l, err = Amf0.ReadNull(b)
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))
	b = []byte{0}
	l, err = Amf0.ReadNull(b)
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfInvalidType))

	b = nil
	objs, l, err = Amf0.ReadObject(b)
	assert.Equal(t, nil, objs)
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfTooShort))
	b = []byte{0}
	objs, l, err = Amf0.ReadObject(b)
	assert.Equal(t, nil, objs)
	assert.Equal(t, 0, l)
	assert.Equal(t, true, errors.Is(err, base.ErrAmfInvalidType))

	defer func() {
		recover()
	}()
	objs = []ObjectPair{
		{Key: "key", Value: []byte{1}},
	}
	_ = Amf0.WriteObject(mw, objs)
}

func TestAmf0_ReadObject_case1(t *testing.T) {
	// 易推流App iOS端

	golden := "030008666c61736856657202001f464d4c452f332e302028636f6d70617469626c653b20464d53632f312e3029000c6361706162696c697469657300406de00000000000000b766964656f436f6465637300406000000000000000036170700200046c697665000673776655726c05000466706164010000077061676555726c05000d766964656f46756e6374696f6e003ff0000000000000000e6f626a656374456e636f64696e67060005746355726c02001a72746d703a2f2f3139322e3136382e33342e3135392f6c697665000b617564696f436f64656373004090000000000000000009"

	b, _ := hex.DecodeString(golden)
	opa, i, err := Amf0.ReadObject(b)
	assert.Equal(t, nil, err)
	assert.Equal(t, 231, i)
	assert.Equal(t, "666c6173685665723a20464d4c452f332e302028636f6d70617469626c653b20464d53632f312e30290a6361706162696c69746965733a203233390a766964656f436f646563733a203132380a6170703a206c6976650a667061643a2066616c73650a766964656f46756e6374696f6e3a20310a746355726c3a2072746d703a2f2f3139322e3136382e33342e3135392f6c6976650a617564696f436f646563733a20313032340a", hex.EncodeToString([]byte(opa.DebugString())))
	//nazalog.Debugf("%+v", opa)
}

func BenchmarkAmf0_ReadObject(b *testing.B) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{Key: "air", Value: 3},
		{Key: "ban", Value: "cat"},
		{Key: "dog", Value: true},
	}
	_ = Amf0.WriteObject(out, objs)
	for i := 0; i < b.N; i++ {
		_, _, _ = Amf0.ReadObject(out.Bytes())
	}
}

func BenchmarkAmf0_WriteObject(b *testing.B) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{Key: "air", Value: 3},
		{Key: "ban", Value: "cat"},
		{Key: "dog", Value: true},
	}
	for i := 0; i < b.N; i++ {
		_ = Amf0.WriteObject(out, objs)
	}
}
