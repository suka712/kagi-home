package cli

import (
	"bufio"
	"io"
	"strings"
)

func bufioReader(r io.Reader) *bufio.Reader {
	return bufio.NewReader(r)
}

func trimNewline(s string) string {
	return strings.TrimSpace(s)
}
