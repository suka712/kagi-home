package store

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// newID returns a new sortable, collision-free ID.
func newID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// slugify turns a title into a filesystem-friendly slug.
func slugify(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "untitled"
	}
	if len(slug) > 60 {
		slug = strings.Trim(slug[:60], "-")
	}
	return slug
}

// uniquePath returns dir/slug.md, disambiguating with a short random suffix
// if a file with that name already exists.
func uniquePath(dir, slug string) (string, error) {
	path := dir + "/" + slug + ".md"
	if !exists(path) {
		return path, nil
	}
	for i := 0; i < 20; i++ {
		suffix, err := randomSuffix(4)
		if err != nil {
			return "", err
		}
		candidate := dir + "/" + slug + "-" + suffix + ".md"
		if !exists(candidate) {
			return candidate, nil
		}
	}
	return "", errTooManyCollisions
}

func randomSuffix(n int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b), nil
}
