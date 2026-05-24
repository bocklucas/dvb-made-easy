package storage

import (
	"regexp"
	"strings"
)

var defaultPattern = regexp.MustCompile(`^backup-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}\.tar\.gz(\.gpg)?$`)

func MatchBackupFilename(name string) (match bool, encrypted bool) {
	return MatchBackupPattern(name, nil)
}

func MatchBackupPattern(name string, pattern *regexp.Regexp) (match bool, encrypted bool) {
	if pattern == nil {
		pattern = defaultPattern
	}
	m := pattern.FindStringSubmatch(name)
	if m == nil {
		return false, false
	}
	for _, sub := range m[1:] {
		if sub == ".gpg" {
			return true, true
		}
	}
	return true, false
}

var strftimeReplacer = strings.NewReplacer(
	"%Y", `\d{4}`,
	"%m", `\d{2}`,
	"%d", `\d{2}`,
	"%H", `\d{2}`,
	"%M", `\d{2}`,
	"%S", `\d{2}`,
)

func CompileBackupPattern(filenameFormat string) (*regexp.Regexp, error) {
	s := filenameFormat
	s = strings.ReplaceAll(s, "{{ .Extension }}", "tar.gz")
	s = strings.ReplaceAll(s, "{{.Extension}}", "tar.gz")
	s = strftimeReplacer.Replace(s)
	s = escapeNonPattern(s)
	s = `^` + s + `(\.gpg)?$`
	return regexp.Compile(s)
}

func CompileBackupPatternWithCapture(filenameFormat string) (*regexp.Regexp, error) {
	s := filenameFormat
	s = strings.ReplaceAll(s, "{{ .Extension }}", "tar.gz")
	s = strings.ReplaceAll(s, "{{.Extension}}", "tar.gz")

	// Find the range of strftime placeholders to wrap in a capture group.
	firstIdx := -1
	lastEnd := -1
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '%' {
			switch s[i+1] {
			case 'Y', 'm', 'd', 'H', 'M', 'S':
				if firstIdx == -1 {
					firstIdx = i
				}
				lastEnd = i + 2
			}
		}
	}

	if firstIdx == -1 {
		return CompileBackupPattern(filenameFormat)
	}

	before := s[:firstIdx]
	tsPart := s[firstIdx:lastEnd]
	after := s[lastEnd:]

	before = escapeNonPattern(strftimeReplacer.Replace(before))
	tsRegex := strftimeReplacer.Replace(tsPart)
	after = escapeNonPattern(strftimeReplacer.Replace(after))

	full := `^` + before + `(` + tsRegex + `)` + after + `(\.gpg)?$`
	return regexp.Compile(full)
}

func ExtractTimestamp(filename string, pattern *regexp.Regexp) (string, bool) {
	m := pattern.FindStringSubmatch(filename)
	if m == nil || len(m) < 2 {
		return "", false
	}
	return m[1], true
}

func escapeNonPattern(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) && s[i+1] == 'd' {
			b.WriteString(`\d`)
			i++
			continue
		}
		if c == '.' {
			b.WriteString(`\.`)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
