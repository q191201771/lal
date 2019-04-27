package bele

// assume local is `le`

func BeUInt24(p []byte) (ret uint32) {
	ret = 0
	ret |= uint32(p[0]) << 16
	ret |= uint32(p[1]) << 8
	ret |= uint32(p[2])
	return
}

func BeUInt16(p []byte) uint16 {
	return (uint16(p[0]) << 8) | uint16(p[1])
}
