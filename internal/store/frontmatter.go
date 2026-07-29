package store

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const frontmatterDelim = "---"

// splitFrontmatter splits a file's contents into the YAML frontmatter block
// and the markdown body that follows it.
func splitFrontmatter(data []byte) (fm []byte, body string, err error) {
	s := string(data)
	prefix := frontmatterDelim + "\n"
	if !bytes.HasPrefix(data, []byte(prefix)) {
		return nil, "", fmt.Errorf("missing frontmatter delimiter")
	}
	rest := s[len(prefix):]
	end := bytes.Index([]byte(rest), []byte("\n"+frontmatterDelim))
	if end == -1 {
		return nil, "", fmt.Errorf("unterminated frontmatter block")
	}
	fmBlock := rest[:end]
	afterDelim := rest[end+len("\n"+frontmatterDelim):]
	afterDelim = trimLeadingNewline(afterDelim)
	return []byte(fmBlock), afterDelim, nil
}

func trimLeadingNewline(s string) string {
	if len(s) > 0 && s[0] == '\n' {
		return s[1:]
	}
	if len(s) > 1 && s[0] == '\r' && s[1] == '\n' {
		return s[2:]
	}
	return s
}

// readFrontmatterFile reads a file and unmarshals its frontmatter into meta,
// returning the markdown body.
func readFrontmatterFile(path string, meta any) (body string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if err := yaml.Unmarshal(fm, meta); err != nil {
		return "", fmt.Errorf("%s: parsing frontmatter: %w", path, err)
	}
	return body, nil
}

// writeFrontmatterFile writes meta as a YAML frontmatter block followed by
// body to path.
func writeFrontmatterFile(path string, meta any, body string) error {
	fm, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(frontmatterDelim + "\n")
	buf.Write(fm)
	buf.WriteString(frontmatterDelim + "\n")
	if body != "" {
		buf.WriteString("\n")
		buf.WriteString(body)
		if body[len(body)-1] != '\n' {
			buf.WriteString("\n")
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
