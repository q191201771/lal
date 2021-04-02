package rtmp

type TagHeader struct {
	Type      uint8  // type
	DataSize  uint32 // body大小，不包含 header 和 prev tag size 字段 3byte
	Timestamp uint32 // 绝对时间戳，单位毫秒 4byte
	StreamID  uint32 // always 0 3byte
}
