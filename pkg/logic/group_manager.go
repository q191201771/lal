// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

// TODO
// - 现有逻辑重构至当前模块中【DONE】
// - server_manger接入当前模块，替换掉原来的map【DONE】
// - 重构整理使用server_manager的地方【DONE】
// - 实现appName逻辑的IGroupManager
// - 配置文件或var.go中增加选取具体IGroupManager实现的开关
// - 去除配置文件中一部分的url_pattern
// - 更新相应的文档：本文件注释，server_manager等中原有关于appName的注释，配置文件文档，流地址列表文档
// - 其他
//     - 前面没有appname，后面又有了，可以更新一次

// ---------------------------------------------------------------------------------------------------------------------

// OnIterateGroup
//
// @return 如果返回false，则删除这个元素
//
type OnIterateGroup func(appName string, streamName string, group *Group) bool

// IGroupManager
//
// 封装管理Group的容器
// 管理流标识（appName，streamName）与Group的映射关系
//
type IGroupManager interface {
	// GetOrCreateGroup
	//
	// @param appName 如果没有，可以为""
	//
	// @return createFlag 如果为false表示group之前就存在，如果为true表示为当前新创建
	//
	GetOrCreateGroup(appName string, streamName string) (group *Group, createFlag bool)

	// GetGroup
	//
	// @return 如果不存在，则返回nil
	//
	GetGroup(appName string, streamName string) *Group

	Iterate(onIterateGroup OnIterateGroup)
	Len() int
}

// SimpleGroupManager 忽略appName，只使用streamName
//
type SimpleGroupManager struct {
	groups map[string]*Group // streamName -> Group
}

func NewSimpleGroupManager() *SimpleGroupManager {
	return &SimpleGroupManager{
		groups: make(map[string]*Group),
	}
}

func (s *SimpleGroupManager) GetOrCreateGroup(appName string, streamName string) (group *Group, createFlag bool) {
	g, ok := s.groups[streamName]
	if !ok {
		g = NewGroup(appName, streamName)
		s.groups[streamName] = g
		return g, true
	}
	return g, false
}

func (s *SimpleGroupManager) GetGroup(appName string, streamName string) *Group {
	g, ok := s.groups[streamName]
	if !ok {
		return nil
	}
	return g
}

func (s *SimpleGroupManager) Iterate(onIterateGroup OnIterateGroup) {
	for streamName, group := range s.groups {
		if !onIterateGroup("", streamName, group) {
			delete(s.groups, streamName)
		}
	}
}

func (s *SimpleGroupManager) Len() int {
	return len(s.groups)
}

// ComplexGroupManager
//
// 背景：
//   有的协议需要结合appName和streamName作为流唯一标识（比如rtmp，httpflv，httpts）
//   有的协议不需要appName，只使用streamName作为流唯一标识（比如rtsp？）
// 目标：
//   有appName的协议，需要参考appName
//   没appName的协议，需要和有appName的协议互通
// 注意：
//   当以上两种类型的协议混用时，系统使用者应避免第二种协议的streamName，在第一种协议中存在相同的streamName，但是appName不止一个
//   这种情况下，内部无法知道该如何对应
// 实现：
//   group可能由第一种协议创建，也可能由第二种协议创建

//type GroupManager struct {
//	// 注意，一个group只可能在一个容器中
//	onlyStreamNameGroups map[string]*Group // streamName -> Group，注意，并不是全量，有appname的group有可能在另外一个容器中
//	appNameStreamNameGroups map[string]map[string]*Group // appName -> streamName -> Group
//}
//
//func NewGroupManager() *GroupManager {
//	return &GroupManager{
//		onlyStreamNameGroups: make(map[string]*Group),
//		appNameStreamNameGroups:make(map[string]map[string]*Group),
//	}
//}
//
//func (gm *GroupManager) GetOrCreateGroup(appName string, streamName string) *Group {
//	if appName == "" {
//		group, exist := gm.onlyStreamNameGroups[streamName]
//		if exist {
//			return group
//		}
//		// 虽然没有appName，也有可能在appNameStreamNameGroups中
//		//
//		// 注意，此时有可能不同appName的容器里都有对应这个streamName的group，但是程序已没法区分，系统使用者应规范流名称避免出现这种问题
//		for _, m := range gm.appNameStreamNameGroups {
//			group, exist := m[streamName]
//			if exist {
//				return group
//			}
//		}
//		// 两个容器都没找到
//		group = NewGroup(appName, streamName)
//		gm.onlyStreamNameGroups[streamName] = group
//		return group
//	} else { // appName存在
//		m, exist := gm.appNameStreamNameGroups[appName]
//		if exist {
//			group, exist := m[streamName]
//			if exist {
//				return group
//			}
//			// 在到了appName对应的容器，但是没有streamName对应的group
//			group = NewGroup(appName, streamName)
//			m[streamName] = group
//			return group
//		}
//		// 虽然有appName，也有可能在onlyStreamNameGroups中，我们尝试一下
//		group, exist := gm.onlyStreamNameGroups[streamName]
//		if exist {
//			return group
//		}
//
//	}
//
//	return nil
//}
//
//func (gm *GroupManager) GetGroup(appName string, streamName string) *Group {
//	return nil
//}
//
//func (gm *GroupManager) Iterate() {
//	panic(0)
//}
//
//func (gm *GroupManager) Delete() {
//	panic(0)
//}
