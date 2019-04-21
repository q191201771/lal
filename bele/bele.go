package bele

func BeUInt24(p []byte) (ret uint32) {
	ret = 0
	ret |= uint32(p[0]) << 16
	ret |= uint32(p[1]) << 8
	ret |= uint32(p[2])
	return
}
