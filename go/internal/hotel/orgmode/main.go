package orgmode

import (
	"fmt"
	"strings"
)

// Document is a parsed orgmode file. It contains an optional preamble (text
// before the first heading) and a sequence of top-level headings.
type Document struct {
	Preamble string
	Headings []Heading
}

// Heading is a single org heading node. Nesting is represented by Level
// (1 = *, 2 = **, etc.). Subheadings are children.
type Heading struct {
	Level      int
	Keyword    string // TODO, DONE, etc.
	Title      string
	Tags       []string
	Properties Properties
	Body       string
	Children   []Heading
}

// Property is a single key-value pair from a :PROPERTIES: drawer.
type Property struct {
	Key   string
	Value string
}

// Properties is the :PROPERTIES: drawer content as ordered key-value pairs.
type Properties []Property

// Get returns the value for a key and whether it was found.
func (props Properties) Get(key string) (string, bool) {
	for _, p := range props {
		if p.Key == key {
			return p.Value, true
		}
	}

	return "", false
}

// Set sets a key-value pair, replacing an existing key or appending.
func (props *Properties) Set(key, value string) {
	for i, p := range *props {
		if p.Key == key {
			(*props)[i].Value = value
			return
		}
	}

	*props = append(*props, Property{Key: key, Value: value})
}

// Parse reads orgmode text and returns a Document. This is a minimal parser
// sufficient for round-tripping dodder zettels through orgmode: it handles
// headings, TODO keywords, tags, property drawers, and body text.
func Parse(text string) (document Document, err error) {
	// Normalize Windows line endings before splitting.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	var preambleLines []string
	var headings []Heading
	var stack []*Heading

	i := 0
	for i < len(lines) {
		line := lines[i]

		level, rest := parseHeadingPrefix(line)
		if level == 0 {
			if len(stack) == 0 {
				preambleLines = append(preambleLines, line)
			} else {
				current := stack[len(stack)-1]
				if current.Body != "" {
					current.Body += "\n"
				}
				current.Body += line
			}
			i++
			continue
		}

		heading := Heading{Level: level}
		heading.Keyword, heading.Title, heading.Tags = parseHeadingLine(rest)

		i++

		// Parse property drawer if present.
		if i < len(lines) {
			heading.Properties, i = parsePropertyDrawer(lines, i)
		}

		// Unwind stack to find parent.
		for len(stack) > 0 && stack[len(stack)-1].Level >= level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			headings = append(headings, heading)
			stack = []*Heading{&headings[len(headings)-1]}
		} else {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, heading)
			stack = append(stack, &parent.Children[len(parent.Children)-1])
		}
	}

	document.Preamble = strings.Join(preambleLines, "\n")
	document.Headings = headings

	return document, nil
}

// Serialize renders a Document back to orgmode text.
func Serialize(document Document) string {
	var builder strings.Builder

	if document.Preamble != "" {
		builder.WriteString(document.Preamble)
		if len(document.Headings) > 0 {
			builder.WriteString("\n")
		}
	}

	for idx, heading := range document.Headings {
		if idx > 0 || document.Preamble != "" {
			builder.WriteString("\n")
		}
		serializeHeading(&builder, &heading)
	}

	return builder.String()
}

// MakeHeading constructs a Heading from dodder object fields. This is the
// decompile path: dodder -> orgmode.
func MakeHeading(
	title string,
	tags []string,
	body string,
	properties Properties,
) Heading {
	heading := Heading{
		Level:      1,
		Title:      title,
		Tags:       tags,
		Properties: properties,
		Body:       body,
	}

	return heading
}

func serializeHeading(builder *strings.Builder, heading *Heading) {
	// Stars.
	builder.WriteString(strings.Repeat("*", heading.Level))
	builder.WriteString(" ")

	// TODO keyword.
	if heading.Keyword != "" {
		builder.WriteString(heading.Keyword)
		builder.WriteString(" ")
	}

	// Title.
	builder.WriteString(heading.Title)

	// Tags.
	if len(heading.Tags) > 0 {
		builder.WriteString(" :")
		builder.WriteString(strings.Join(heading.Tags, ":"))
		builder.WriteString(":")
	}

	builder.WriteString("\n")

	// Property drawer.
	if len(heading.Properties) > 0 {
		builder.WriteString(":PROPERTIES:\n")
		for _, prop := range heading.Properties {
			fmt.Fprintf(builder, ":%s: %s\n", prop.Key, prop.Value)
		}
		builder.WriteString(":END:\n")
	}

	// Body.
	if heading.Body != "" {
		builder.WriteString(heading.Body)
		builder.WriteString("\n")
	}

	// Children.
	for _, child := range heading.Children {
		serializeHeading(builder, &child)
	}
}

// parseHeadingPrefix checks if a line starts with one or more '*' followed
// by a space. Returns the heading level and the remainder of the line.
func parseHeadingPrefix(line string) (level int, rest string) {
	if len(line) == 0 || line[0] != '*' {
		return 0, line
	}

	for level < len(line) && line[level] == '*' {
		level++
	}

	if level >= len(line) || line[level] != ' ' {
		return 0, line
	}

	return level, line[level+1:]
}

// parseHeadingLine extracts the TODO keyword, title, and tags from the text
// after the stars prefix.
func parseHeadingLine(rest string) (keyword, title string, tags []string) {
	// Check for TODO keyword at the start.
	for _, kw := range []string{"TODO", "DONE", "NEXT", "WAITING", "CANCELLED"} {
		if strings.HasPrefix(rest, kw+" ") || rest == kw {
			keyword = kw
			rest = strings.TrimPrefix(rest, kw)
			rest = strings.TrimLeft(rest, " ")
			break
		}
	}

	// Check for tags at the end: :tag1:tag2:
	if idx := strings.LastIndex(rest, " :"); idx >= 0 {
		tagPart := rest[idx+2:]
		if strings.HasSuffix(tagPart, ":") && !strings.Contains(tagPart[:len(tagPart)-1], " ") {
			tagStr := tagPart[:len(tagPart)-1]
			tags = strings.Split(tagStr, ":")
			rest = strings.TrimRight(rest[:idx], " ")
		}
	}

	title = rest
	return keyword, title, tags
}

// parsePropertyDrawer reads the :PROPERTIES:...:END: block starting at line
// index i. Returns the properties and the next line index after :END:.
func parsePropertyDrawer(lines []string, i int) (Properties, int) {
	trimmed := strings.TrimSpace(lines[i])
	if trimmed != ":PROPERTIES:" {
		return nil, i
	}

	var props Properties
	i++

	for i < len(lines) {
		trimmed = strings.TrimSpace(lines[i])
		i++

		if trimmed == ":END:" {
			break
		}

		// Parse :KEY: VALUE
		if len(trimmed) > 1 && trimmed[0] == ':' {
			rest := trimmed[1:]
			colonIdx := strings.Index(rest, ":")
			if colonIdx > 0 {
				key := rest[:colonIdx]
				value := strings.TrimSpace(rest[colonIdx+1:])
				props = append(props, Property{Key: key, Value: value})
			}
		}
	}

	return props, i
}
