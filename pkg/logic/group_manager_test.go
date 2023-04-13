// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"testing"

	"github.com/q191201771/naza/pkg/assert"
)

type mockGroupCreator struct {
}

func (m *mockGroupCreator) CreateGroup(appName, streamName string) *Group {
	var config Config
	return NewGroup(appName, streamName, &config, GroupOption{}, nil)
}

var mgc = &mockGroupCreator{}

func TestGroupManager(t *testing.T) {
	var (
		sgm IGroupManager
		sg0 *Group
		sg1 *Group
		sg2 *Group
		sg3 *Group
		sg4 *Group

		cgm IGroupManager
		cg0 *Group
		cg1 *Group
		cg2 *Group
		cg3 *Group
		cg4 *Group

		createFlag bool
	)

	sgm = NewSimpleGroupManager(mgc)
	cgm = NewComplexGroupManager(mgc)

	// (为空时)获取
	// 获取到nil
	// 表现一致
	{
		sg0 = sgm.GetGroup("app1", "stream1")
		assert.Equal(t, nil, sg0)
		assert.Equal(t, 0, sgm.Len())

		cg0 = cgm.GetGroup("app1", "stream1")
		assert.Equal(t, nil, cg0)
		assert.Equal(t, 0, cgm.Len())
	}

	// (为空时)创建
	// 创建成功
	// 表现一致
	{
		sg1, createFlag = sgm.GetOrCreateGroup("app1", "stream1")
		assert.IsNotNil(t, sg1)
		assert.Equal(t, true, createFlag)
		assert.Equal(t, 1, sgm.Len())
		sg0 = sgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, sg0 == sg1)

		cg1, createFlag = cgm.GetOrCreateGroup("app1", "stream1")
		assert.IsNotNil(t, cg1)
		assert.Equal(t, true, createFlag)
		assert.Equal(t, 1, cgm.Len())
		cg0 = cgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, cg0 == cg1)

		// cache sg1 -> app1, stream1
		// cache cg1 -> app1, stream1
	}

	// appName和streamName相同
	// 不再创建
	// 表现一致
	{
		sg2, createFlag = sgm.GetOrCreateGroup("app1", "stream1")
		assert.IsNotNil(t, sg2)
		assert.Equal(t, false, createFlag)
		assert.Equal(t, 1, sgm.Len())
		assert.Equal(t, true, sg1 == sg2)
		sg0 = sgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, sg0 == sg1)

		cg2, createFlag = cgm.GetOrCreateGroup("app1", "stream1")
		assert.IsNotNil(t, cg2)
		assert.Equal(t, false, createFlag)
		assert.Equal(t, 1, cgm.Len())
		assert.Equal(t, true, cg1 == cg2)
		cg0 = cgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, cg0 == cg1)

		sg2 = nil
		cg2 = nil
	}

	// appName相同，streamName不同
	// 都创建
	// 表现一致
	{
		sg2, createFlag = sgm.GetOrCreateGroup("app1", "stream2")
		assert.IsNotNil(t, sg2)
		assert.Equal(t, true, createFlag)
		assert.Equal(t, 2, sgm.Len())
		sg0 = sgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, sg0 == sg1)
		sg0 = sgm.GetGroup("app1", "stream2")
		assert.Equal(t, true, sg0 == sg2)

		cg2, createFlag = cgm.GetOrCreateGroup("app1", "stream2")
		assert.IsNotNil(t, cg2)
		assert.Equal(t, true, createFlag)
		assert.Equal(t, 2, cgm.Len())
		assert.Equal(t, true, cg1 != cg2)
		cg0 = cgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, cg0 == cg1)
		cg0 = cgm.GetGroup("app1", "stream2")
		assert.Equal(t, true, cg0 == cg2)

		// cache sg1 -> app1, stream1
		// cache sq2 -> app1, stream2
		//
		// cache cg1 -> app1, stream1
		// cache cq2 -> app1, stream2
	}

	// appName不同，streamName相同
	// SimpleGroupManager不再创建，ComplexGroupManager继续创建
	// 表现不一致
	{
		sg3, createFlag = sgm.GetOrCreateGroup("app2", "stream1")
		assert.IsNotNil(t, sg3)
		assert.Equal(t, false, createFlag)
		assert.Equal(t, 2, sgm.Len())
		assert.Equal(t, true, sg1 == sg3)
		sg0 = sgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, sg0 == sg1)
		sg0 = sgm.GetGroup("app1", "stream2")
		assert.Equal(t, true, sg0 == sg2)
		sg0 = sgm.GetGroup("app2", "stream1")
		assert.Equal(t, true, sg0 == sg3)

		cg3, createFlag = cgm.GetOrCreateGroup("app2", "stream1")
		assert.IsNotNil(t, cg3)
		assert.Equal(t, true, createFlag)
		assert.Equal(t, 3, cgm.Len())
		cg0 = cgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, cg0 == cg1)
		cg0 = cgm.GetGroup("app1", "stream2")
		assert.Equal(t, true, cg0 == cg2)
		cg0 = cgm.GetGroup("app2", "stream1")
		assert.Equal(t, true, cg0 == cg3)

		// cache sg1 -> app1, stream1
		// cache sq2 -> app1, stream2
		// cache sq3 -> app2, stream1 == sq1
		//
		// cache cg1 -> app1, stream1
		// cache cq2 -> app1, stream2
		// cache cq3 -> app2, stream1
	}

	// appName不同，streamName也不同
	// 表现一致
	{
		sg4, createFlag = sgm.GetOrCreateGroup("app3", "stream3")
		assert.IsNotNil(t, sg4)
		assert.Equal(t, true, createFlag)
		assert.Equal(t, 3, sgm.Len())
		assert.Equal(t, true, sg1 == sg3)
		sg0 = sgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, sg0 == sg1)
		sg0 = sgm.GetGroup("app1", "stream2")
		assert.Equal(t, true, sg0 == sg2)
		sg0 = sgm.GetGroup("app2", "stream1")
		assert.Equal(t, true, sg0 == sg3)
		sg0 = sgm.GetGroup("app3", "stream3")
		assert.Equal(t, true, sg0 == sg4)

		cg4, createFlag = cgm.GetOrCreateGroup("app3", "stream3")
		assert.IsNotNil(t, cg4)
		assert.Equal(t, true, createFlag)
		assert.Equal(t, 4, cgm.Len())
		cg0 = cgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, cg0 == cg1)
		cg0 = cgm.GetGroup("app1", "stream2")
		assert.Equal(t, true, cg0 == cg2)
		cg0 = cgm.GetGroup("app2", "stream1")
		assert.Equal(t, true, cg0 == cg3)
		cg0 = cgm.GetGroup("app3", "stream3")
		assert.Equal(t, true, cg0 == cg4)

		// cache sg1 -> app1, stream1
		// cache sq2 -> app1, stream2
		// cache sq3 -> app2, stream1 == sq1
		// cache sq4 -> app3, stream3
		//
		// cache cg1 -> app1, stream1
		// cache cq2 -> app1, stream2
		// cache cq3 -> app2, stream1
		// cache cq4 -> app3, stream3
	}

	// 遍历以及删除
	{
		sgm.Iterate(func(group *Group) bool {
			appName := group.appName
			streamName := group.streamName

			if appName == "app1" && streamName == "stream1" {
				assert.Equal(t, true, group == sg1)
				return true
			} else if appName == "app1" && streamName == "stream2" {
				assert.Equal(t, true, group == sg2)
				// erase
				return false
			} else if appName == "app3" && streamName == "stream3" {
				assert.Equal(t, true, group == sg4)
				return true
			}
			// never reach here
			assert.Equal(t, true, false, appName, streamName)
			return true
		})

		assert.Equal(t, 2, sgm.Len())
		sg0 = sgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, sg0 == sg1)
		sg0 = sgm.GetGroup("app3", "stream3")
		assert.Equal(t, true, sg0 == sg4)

		cgm.Iterate(func(group *Group) bool {
			appName := group.appName
			streamName := group.streamName

			if appName == "app1" && streamName == "stream1" {
				assert.Equal(t, true, group == cg1)
				return true
			} else if appName == "app1" && streamName == "stream2" {
				assert.Equal(t, true, group == cg2)
				// erase
				return false
			} else if appName == "app2" && streamName == "stream1" {
				assert.Equal(t, true, group == cg3)
				return true
			} else if appName == "app3" && streamName == "stream3" {
				assert.Equal(t, true, group == cg4)
				return true
			}
			// never reach here
			assert.Equal(t, true, false, appName, streamName)
			return true
		})

		assert.Equal(t, 3, cgm.Len())
		cg0 = cgm.GetGroup("app1", "stream1")
		assert.Equal(t, true, cg0 == cg1)
		cg0 = cgm.GetGroup("app2", "stream1")
		assert.Equal(t, true, cg0 == cg3)
		cg0 = cgm.GetGroup("app3", "stream3")
		assert.Equal(t, true, cg0 == cg4)
	}

	//----------------------------

	sgm = NewSimpleGroupManager(mgc)
	cgm = NewComplexGroupManager(mgc)

	sg0 = cgm.GetGroup("", "stream1")
	cg0 = cgm.GetGroup("", "stream1")
	assert.Equal(t, nil, sg0)
	assert.Equal(t, nil, cg0)

	sg1, createFlag = sgm.GetOrCreateGroup("", "stream1")
	assert.IsNotNil(t, sg1)
	assert.Equal(t, true, createFlag)
	cg1, createFlag = cgm.GetOrCreateGroup("", "stream1")
	assert.IsNotNil(t, cg1)
	assert.Equal(t, true, createFlag)

	sg0 = sgm.GetGroup("", "stream1")
	cg0 = cgm.GetGroup("", "stream1")
	assert.Equal(t, sg1, sg0)
	assert.Equal(t, cg1, cg0)

	sg0 = sgm.GetGroup("app1", "stream1")
	cg0 = cgm.GetGroup("app1", "stream1")
	assert.Equal(t, sg1, sg0)
	assert.Equal(t, cg1, cg0)

	sgm.Iterate(func(group *Group) bool {
		assert.Equal(t, sg1, group)
		return false
	})
	cgm.Iterate(func(group *Group) bool {
		assert.Equal(t, cg1, group)
		return false
	})
	assert.Equal(t, 0, sgm.Len())
	assert.Equal(t, 0, cgm.Len())

	sg1, createFlag = sgm.GetOrCreateGroup("app1", "stream1")
	assert.IsNotNil(t, sg1)
	assert.Equal(t, true, createFlag)
	cg1, createFlag = cgm.GetOrCreateGroup("app1", "stream1")
	assert.IsNotNil(t, cg1)
	assert.Equal(t, true, createFlag)

	sg0 = sgm.GetGroup("", "stream1")
	cg0 = cgm.GetGroup("", "stream1")
	assert.Equal(t, sg1, sg0)
	assert.Equal(t, cg1, cg0)

	sgm.Iterate(func(group *Group) bool {
		assert.Equal(t, sg1, group)
		return false
	})
	assert.Equal(t, 0, sgm.Len())
	cgm.Iterate(func(group *Group) bool {
		assert.Equal(t, cg1, group)
		return false
	})
	assert.Equal(t, 0, cgm.Len())
}
