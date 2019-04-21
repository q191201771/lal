package httpflv

import (
	"bufio"
	"errors"
	"strings"
)

var fxxkErr = errors.New("fxxk")

var readBufSize = 4096
var writeBufSize = 4096

// return 1st line and other headers with kv format
func parseHttpHeader(r *bufio.Reader) (firstLine string, headers map[string]string, err error) {
	headers = make(map[string]string)

	var line []byte
	var isPrefix bool
	line, isPrefix, err = r.ReadLine()
	if len(line) == 0 || isPrefix || err != nil {
		err = fxxkErr
		return
	}
	firstLine = string(line)

	for {
		line, isPrefix, err = r.ReadLine()
		if len(line) == 0 {
			break
		}
		if isPrefix || err != nil {
			err = fxxkErr
			return
		}
		l := string(line)
		pos := strings.Index(l, ":")
		if pos == -1 {
			err = fxxkErr
			return
		}
		headers[strings.Trim(l[0:pos], " ")] = strings.Trim(l[pos+1:], " ")
	}
	return
}
