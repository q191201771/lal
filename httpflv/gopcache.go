package httpflv

import (
	"bytes"
	"github.com/q191201771/lal/log"
	"sync"
)

type Gop struct {
	raw            []byte
	firstTimestamp uint32
}

type GopCache struct {
	gopNum int

	metadata     *Tag
	avcSeqHeader *Tag
	aacSeqHeader *Tag
	gops         []*Gop // TODO chef: maybe use other container to mock a queue
	mutex        sync.Mutex
}

// gopNum: 0 means only cache metadata, avc seq header, aac seq header
func NewGopCache(gopNum int) *GopCache {
	return &GopCache{
		gopNum: gopNum,
	}
}

func (c *GopCache) Push(tag *Tag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if tag.isMetaData() {
		// TODO chef: will this happen?
		if c.metadata != nil {
			log.Debugf("updating metadata.")
			log.Debug(tag.Header, tag.Raw[tagHeaderSize:])
			log.Debug(c.metadata.Header, c.metadata.Raw[tagHeaderSize:])
			c.clearGop()
		}
		c.metadata = tag
	}
	if tag.isAvcKeySeqHeader() {
		//log.Debug(parseAvcSeqHeader(tag.Raw[tagHeaderSize:]))
		if c.avcSeqHeader == nil {
			c.avcSeqHeader = tag
		} else {
			// TODO chef: compare nessary? if other way to update seq header and handle cache stuff?
			if bytes.Compare(tag.Raw[tagHeaderSize:], c.avcSeqHeader.Raw[tagHeaderSize:]) == 0 {
				// noop
			} else {
				log.Debugf("updating avc seq header.")
				log.Debug(tag.Header, tag.Raw[tagHeaderSize:])
				log.Debug(c.avcSeqHeader.Header, c.avcSeqHeader.Raw[tagHeaderSize:])
				c.clearGop()
				c.avcSeqHeader = tag
			}
		}
	}
	if tag.isAacSeqHeader() {
		if c.aacSeqHeader == nil {
			c.aacSeqHeader = tag
		} else {
			if bytes.Compare(tag.Raw[tagHeaderSize:], c.aacSeqHeader.Raw[tagHeaderSize:]) == 0 {
				// noop
			} else {
				log.Debugf("updating aac seq header.")
				c.clearGop()
				c.aacSeqHeader = tag
			}
		}
		c.aacSeqHeader = tag
	}

	if c.gopNum == 0 {
		return
	}

	if len(c.gops) == 0 {
		if tag.isAvcKeyNalu() {
			gop := &Gop{}
			gop.firstTimestamp = tag.Header.Timestamp
			gop.raw = append(gop.raw, tag.Raw...)
			c.gops = append(c.gops, gop)
			c.syncOldestKeyNaluTimestampToSeqHeader()
		}
	} else {
		if tag.isAvcKeyNalu() {
			gop := &Gop{}
			gop.firstTimestamp = tag.Header.Timestamp
			gop.raw = append(gop.raw, tag.Raw...)
			c.gops = append(c.gops, gop)
			if len(c.gops) > c.gopNum+1 {
				c.gops = c.gops[1:]
				c.syncOldestKeyNaluTimestampToSeqHeader()
			}
		} else {
			c.gops[len(c.gops)-1].raw = append(c.gops[len(c.gops)-1].raw, tag.Raw...)
		}
	}
}

func (c *GopCache) GetWholeThings() (hasKeyFrame bool, res []byte) {
	if tag := c.getMetadata(); tag != nil {
		res = append(res, tag.Raw...)
	}

	avc := c.getAvcSeqHeader()
	aac := c.getAacSeqHeader()
	// TODO chef: if nessary to sort them by timestamp
	if avc != nil && aac != nil {
		if avc.Header.Timestamp <= aac.Header.Timestamp {
			res = append(res, avc.Raw...)
			res = append(res, aac.Raw...)
		} else {
			res = append(res, aac.Raw...)
			res = append(res, avc.Raw...)
		}
	} else if avc != nil && aac == nil {
		res = append(res, avc.Raw...)
	} else if avc == nil && aac != nil {
		res = append(res, aac.Raw...)
	}

	if gops := c.getGops(false); gops != nil {
		res = append(res, gops...)
		log.Debug("cache match.")
		hasKeyFrame = true
	}
	return
}

func (c *GopCache) ClearAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metadata = nil
	c.avcSeqHeader = nil
	c.aacSeqHeader = nil
	c.gops = nil
}

func (c *GopCache) getGops(mustCompleted bool) []byte {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	neededLen := len(c.gops)
	if mustCompleted {
		neededLen--
	}
	if neededLen <= 0 {
		return nil
	}

	var res []byte
	for i := 0; i != neededLen; i++ {
		res = append(res, c.gops[i].raw...)
	}
	return res
}

func (c *GopCache) getMetadata() (res *Tag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.metadata != nil {
		res = c.metadata.cloneTag()
	}
	return
}

func (c *GopCache) getAvcSeqHeader() (res *Tag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.avcSeqHeader != nil {
		res = c.avcSeqHeader.cloneTag()
	}
	return
}

func (c *GopCache) getAacSeqHeader() (res *Tag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.aacSeqHeader != nil {
		res = c.aacSeqHeader.cloneTag()
	}
	return
}

func (c *GopCache) clearGop() {
	log.Debug("clearGop")
	c.gops = nil
}

// TODO chef: if nessary
func (c *GopCache) syncOldestKeyNaluTimestampToSeqHeader() {
	ts := c.gops[0].firstTimestamp
	if c.avcSeqHeader != nil {
		c.avcSeqHeader.Header.Timestamp = ts
	}
	if c.aacSeqHeader != nil {
		c.aacSeqHeader.Header.Timestamp = ts
	}
}
