// Package chunking splits Markdown documents into embedding inputs.
package chunking

import "strings"

const (
	// MaxCharacters approximates a 1,000-token embedding input.
	MaxCharacters = 4000
	// OverlapCharacters retains context across oversized section boundaries.
	OverlapCharacters = 400
)

// Markdown splits a document at headings and bounds oversized sections.
func Markdown(content string) []string {
	var (
		chunks         []string
		line           string
		lines          []string
		section        []string
		sectionContent string
		sections       []string
	)

	lines = strings.Split(content, "\n")
	for _, line = range lines {
		if isHeading(line) && len(section) > 0 {
			sections = append(sections, strings.Join(section, "\n"))
			section = nil
		}

		section = append(section, line)
	}
	if len(section) > 0 {
		sections = append(sections, strings.Join(section, "\n"))
	}

	for _, sectionContent = range sections {
		chunks = append(chunks, splitSection(sectionContent)...)
	}

	return chunks
}

func isHeading(line string) bool {
	var index int

	for index = 0; index < len(line) && index < 6 && line[index] == '#'; index++ {
	}

	return index > 0 && index < len(line) && line[index] == ' '
}

func splitSection(section string) []string {
	var (
		body         string
		bodyRunes    []rune
		chunk        string
		chunks       []string
		end          int
		heading      string
		limit        int
		overlap      int
		sectionRunes []rune
		start        int
	)

	section = strings.TrimSpace(section)
	if section == "" {
		return nil
	}

	sectionRunes = []rune(section)
	if len(sectionRunes) <= MaxCharacters {
		return []string{section}
	}

	heading, body = sectionHeading(section)
	limit = MaxCharacters - len([]rune(heading))
	if heading != "" {
		limit -= 2
	}
	if limit <= OverlapCharacters {
		heading = ""
		body = section
		limit = MaxCharacters
	}

	bodyRunes = []rune(body)
	overlap = OverlapCharacters
	if overlap >= limit {
		overlap = limit / 10
	}
	for start < len(bodyRunes) {
		end = start + limit
		if end > len(bodyRunes) {
			end = len(bodyRunes)
		}

		chunk = strings.TrimSpace(string(bodyRunes[start:end]))
		if heading != "" {
			chunk = heading + "\n\n" + chunk
		}
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(bodyRunes) {
			break
		}

		start = end - overlap
	}

	return chunks
}

func sectionHeading(section string) (string, string) {
	var (
		firstLine string
		lineEnd   int
	)

	lineEnd = strings.IndexByte(section, '\n')
	if lineEnd < 0 {
		return "", section
	}

	firstLine = section[:lineEnd]
	if !isHeading(firstLine) {
		return "", section
	}

	return firstLine, strings.TrimSpace(section[lineEnd+1:])
}
