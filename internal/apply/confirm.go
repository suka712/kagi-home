// Package apply implements the shared propose -> preview -> confirm ->
// apply pipeline used by every `hp ai *` command: nothing an AI proposes
// touches the vault until the user has seen and accepted it (or passed
// --yes).
package apply

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Confirmer struct {
	reader  *bufio.Reader
	out     io.Writer
	autoYes bool
}

func NewConfirmer(in io.Reader, out io.Writer, autoYes bool) *Confirmer {
	return &Confirmer{reader: bufio.NewReader(in), out: out, autoYes: autoYes}
}

// Confirm shows preview and asks the user to accept, skip, or quit
// (skipping everything remaining). With autoYes, accepts without prompting.
func (c *Confirmer) Confirm(preview string) (accept, quit bool, err error) {
	fmt.Fprint(c.out, preview)
	if c.autoYes {
		fmt.Fprintln(c.out, "-> auto-accepted (--yes)")
		return true, false, nil
	}
	fmt.Fprint(c.out, "Apply this? [y/N/q(uit)] ")
	line, err := c.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, false, nil
	case "q", "quit":
		return false, true, nil
	default:
		return false, false, nil
	}
}
