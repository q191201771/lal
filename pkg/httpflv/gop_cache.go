// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

/*
// v1.0.0 版本之前不提供 httpflv 功能

import (
	"bytes"
	log "github.com/q191201771/naza/pkg/nazalog"
	"sync"
)

type GOP struct {
	tags []*Tag
	//raw            []byte
	firstTimestamp uint32
}

type GOPCache struct {
	gopNum int

	metadata     *Tag
	avcSeqHeader *Tag
	aacSeqHeader *Tag
	gops         []*GOP // TODO chef: maybe use other container to mock a queue
	mutex        sync.Mutex
}

// gopNum: 0 means only cache metadata, avc seq header, aac seq header
func NewGOPCache(gopNum int) *GOPCache {
	return &GOPCache{
		gopNum: gopNum,
	}
}

func (c *GOPCache) Push(tag *Tag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if tag.IsMetadata() {
		// TODO chef: will this happen?
		if c.metadata != nil {
			log.Debugf("updating metadata.")
			log.Debug(tag.Header, tag.Raw[TagHeaderSize:])
			log.Debug(c.metadata.Header, c.metadata.Raw[TagHeaderSize:])
			c.clearGOP()
		}
		c.metadata = tag
	}
	if tag.IsAVCKeySeqHeader() {
		if c.avcSeqHeader == nil {
			c.avcSeqHeader = tag
		} else {
			// TODO chef: compare nessary? if other way to update seq header and handle cache stuff?
			if bytes.Compare(tag.Raw[TagHeaderSize:], c.avcSeqHeader.Raw[TagHeaderSize:]) == 0 {
				// noop
			} else {
				log.Debugf("updating avc seq header.")
				log.Debug(tag.Header, tag.Raw[TagHeaderSize:])
				log.Debug(c.avcSeqHeader.Header, c.avcSeqHeader.Raw[TagHeaderSize:])
				c.clearGOP()
				c.avcSeqHeader = tag
			}
		}
	}
	if tag.IsAACSeqHeader() {
		if c.aacSeqHeader == nil {
			c.aacSeqHeader = tag
		} else {
			if bytes.Compare(tag.Raw[TagHeaderSize:], c.aacSeqHeader.Raw[TagHeaderSize:]) == 0 {
				// noop
			} else {
				log.Debugf("updating aac seq header.")
				c.clearGOP()
				c.aacSeqHeader = tag
			}
		}
		c.aacSeqHeader = tag
	}

	if c.gopNum == 0 {
		return
	}

	if len(c.gops) == 0 {
		if tag.IsAVCKeyNalu() {
			gop := &GOP{}
			gop.firstTimestamp = tag.Header.Timestamp
			gop.tags = append(gop.tags, tag)
			c.gops = append(c.gops, gop)
			c.syncOldestKeyNaluTimestampToSeqHeader()
		}
	} else {
		if tag.IsAVCKeyNalu() {
			gop := &GOP{}
			gop.firstTimestamp = tag.Header.Timestamp
			gop.tags = append(gop.tags, tag)
			c.gops = append(c.gops, gop)
			if len(c.gops) > c.gopNum+1 {
				c.gops = c.gops[1:]
				c.syncOldestKeyNaluTimestampToSeqHeader()
			}
		} else {
			c.gops[len(c.gops)-1].tags = append(c.gops[len(c.gops)-1].tags, tag)
		}
	}
}

func (c *GOPCache) WriteWholeThings(writer Writer) (hasKeyFrame bool) {
	if tag := c.getMetadata(); tag != nil {
		writer.WriteTag(tag)
	}

	avc := c.getAVCSeqHeader()
	aac := c.getAACSeqHeader()
	// TODO chef: if nessary to sort them by timestamp
	if avc != nil && aac != nil {
		if avc.Header.Timestamp <= aac.Header.Timestamp {
			writer.WriteTag(avc)
			writer.WriteTag(aac)
		} else {
			writer.WriteTag(aac)
			writer.WriteTag(avc)
		}
	} else if avc != nil && aac == nil {
		writer.WriteTag(avc)
	} else if avc == nil && aac != nil {
		writer.WriteTag(aac)
	}
	c.writeGOPs(writer, false)
	return
}

func (c *GOPCache) ClearAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metadata = nil
	c.avcSeqHeader = nil
	c.aacSeqHeader = nil
	c.gops = nil
}

func (c *GOPCache) writeGOPs(write Writer, mustCompleted bool) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	neededLen := len(c.gops)
	if mustCompleted {
		neededLen--
	}
	if neededLen <= 0 {
		return false
	}

	for i := 0; i != neededLen; i++ {
		for j := 0; j != len(c.gops[i].tags); j++ {
			write.WriteTag(c.gops[i].tags[j])
		}
	}
	return true
}

func (c *GOPCache) getMetadata() (res *Tag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.metadata != nil {
		res = c.metadata.cloneTag()
	}
	return
}

func (c *GOPCache) getAVCSeqHeader() (res *Tag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.avcSeqHeader != nil {
		res = c.avcSeqHeader.cloneTag()
	}
	return
}

func (c *GOPCache) getAACSeqHeader() (res *Tag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.aacSeqHeader != nil {
		res = c.aacSeqHeader.cloneTag()
	}
	return
}

func (c *GOPCache) clearGOP() {
	log.Debug("clearGOP")
	c.gops = nil
}

// TODO chef: if nessary
func (c *GOPCache) syncOldestKeyNaluTimestampToSeqHeader() {
	ts := c.gops[0].firstTimestamp
	if c.avcSeqHeader != nil {
		c.avcSeqHeader.Header.Timestamp = ts
	}
	if c.aacSeqHeader != nil {
		c.aacSeqHeader.Header.Timestamp = ts
	}
}

*/
