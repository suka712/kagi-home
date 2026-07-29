package store

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"hp/internal/model"
)

func (s *Store) CreateContextDoc(title, body string) (*model.ContextDoc, error) {
	doc := &model.ContextDoc{Title: title, Created: time.Now(), Body: body}
	path, err := uniquePath(s.contextDir(), slugify(title))
	if err != nil {
		return nil, err
	}
	doc.Path = path
	if err := writeFrontmatterFile(path, doc, doc.Body); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Store) ListContextDocs() ([]*model.ContextDoc, error) {
	paths, err := listMarkdownFiles(s.contextDir())
	if err != nil {
		return nil, err
	}
	docs := make([]*model.ContextDoc, 0, len(paths))
	for _, p := range paths {
		var d model.ContextDoc
		body, err := readFrontmatterFile(p, &d)
		if err != nil {
			return nil, err
		}
		d.Body = body
		d.Path = p
		docs = append(docs, &d)
	}
	return docs, nil
}

// AppendToContextDoc appends a section to an existing context doc (creating
// it if it doesn't exist yet), identified by title. Used by `hp ai ask` to
// log Q&A into context/qa-log.md.
func (s *Store) AppendToContextDoc(title, section string) error {
	docs, err := s.ListContextDocs()
	if err != nil {
		return err
	}
	for _, d := range docs {
		if d.Title == title {
			d.Body = d.Body + "\n" + section
			return writeFrontmatterFile(d.Path, d, d.Body)
		}
	}
	_, err = s.CreateContextDoc(title, section)
	return err
}

// SaveContextFile copies an original uploaded file (e.g. a PDF) verbatim
// into context/files/ for reference. It is not parsed or fed to the AI
// directly; use CreateContextDoc for the text that should be.
func (s *Store) SaveContextFile(srcPath string) (string, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dest := filepath.Join(s.filesDir(), filepath.Base(srcPath))
	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return dest, nil
}
