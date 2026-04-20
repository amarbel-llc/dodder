package haustoria_orgmode

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	orgpeg "code.linenisgreat.com/dodder/go/internal/hotel/orgmode_peg"
	"github.com/google/uuid"
)

// ParsedHeading holds per-heading info extracted from a langlang parse tree.
// All positions are byte offsets into the original input buffer.
type ParsedHeading struct {
	// HeadingStart and HeadingEnd are the byte range of this heading,
	// from '*' at the start of its line through the end of its body.
	HeadingStart int
	HeadingEnd   int

	// Title is the heading's title text (after stars/keyword, before tags).
	Title string

	// Tags are parsed from the heading's :tag1:tag2: suffix on the same
	// line as the title.
	Tags []string

	// BodyStart and BodyEnd are the byte range of this heading's body
	// content (everything after the property drawer or after planning
	// lines if no drawer, up to the next heading).
	BodyStart int
	BodyEnd   int

	// DrawerEndInsertPos is the byte offset where a new property line
	// should be inserted to extend an existing :PROPERTIES: drawer:
	// just before the ':END:' line. Zero if no drawer exists.
	DrawerEndInsertPos int

	// InsertNewDrawerAt is the byte offset where a brand-new property
	// drawer should be inserted when the heading has no drawer. This is
	// the byte position immediately after the heading line's EOL (or
	// after the last planning line if any).
	// Only meaningful when DrawerEndInsertPos is zero.
	InsertNewDrawerAt int

	// ID is the value of the :ID: property if the drawer contains one,
	// or "" if absent or if there is no drawer.
	ID string

	// DodderID is the value of the :DODDER_ID: property if present.
	DodderID string
}

// synthesizeHeadingFromContent produces a single ParsedHeading for files
// that contain no top-level headings, so heading-less notes still surface
// as externally-visible zettels. The ID is derived from the file stem
// (stable across reads without mutating the file) and the title is the
// first non-empty line.
func synthesizeHeadingFromContent(
	content []byte,
	filePath string,
) (ParsedHeading, bool) {
	id := fileExternalId(filePath)
	if id == "" {
		return ParsedHeading{}, false
	}

	titleStart, titleEnd, bodyStart := firstNonEmptyLineBounds(content)
	if titleStart == titleEnd {
		return ParsedHeading{}, false
	}

	title := strings.TrimRight(
		string(content[titleStart:titleEnd]),
		" \t\r",
	)

	return ParsedHeading{
		Title:     title,
		BodyStart: bodyStart,
		BodyEnd:   len(content),
		ID:        id,
	}, true
}

// firstNonEmptyLineBounds returns byte offsets for the first non-empty line
// and the start of the line after it. If no non-empty line exists, both
// bounds are len(content).
func firstNonEmptyLineBounds(content []byte) (start, end, next int) {
	pos := 0
	for pos < len(content) {
		lineEnd := bytes.IndexByte(content[pos:], '\n')
		var lineStop int
		if lineEnd < 0 {
			lineStop = len(content)
		} else {
			lineStop = pos + lineEnd
		}

		line := content[pos:lineStop]
		if len(bytes.TrimSpace(line)) > 0 {
			nextLine := lineStop
			if lineEnd >= 0 {
				nextLine = lineStop + 1
			}
			return pos, lineStop, nextLine
		}

		if lineEnd < 0 {
			break
		}
		pos = lineStop + 1
	}
	return len(content), len(content), len(content)
}

// parseFile parses orgmode content and returns one entry per top-level
// heading in source order.
func parseFile(content []byte) (headings []ParsedHeading, err error) {
	parser := orgpeg.NewParser()
	parser.SetInput(content)

	tree, parseErr := parser.Parse()
	if parseErr != nil {
		return nil, fmt.Errorf("orgmode parse: %w", parseErr)
	}

	root, ok := tree.Root()
	if !ok {
		return nil, nil
	}

	tree.Visit(root, func(id orgpeg.NodeID) bool {
		if tree.Type(id) != orgpeg.NodeType_Node {
			return true
		}
		if tree.Name(id) != "Heading" {
			return true
		}
		headings = append(headings, extractHeading(tree, id, content))
		return false // don't recurse into nested headings
	})

	return headings, nil
}

// extractHeading walks the children of a Heading node and fills out a
// ParsedHeading from the byte spans.
func extractHeading(
	tree orgpeg.Tree,
	headingID orgpeg.NodeID,
	content []byte,
) ParsedHeading {
	headingSpan := tree.Span(headingID)
	heading := ParsedHeading{
		HeadingStart: headingSpan.Start.Cursor,
		HeadingEnd:   headingSpan.End.Cursor,
	}

	var (
		headingLineEOL int
		planningEnd    int
		drawerStart    int
		drawerEnd      int
		hasDrawer      bool
	)

	tree.Visit(headingID, func(id orgpeg.NodeID) bool {
		if id == headingID {
			return true
		}
		if tree.Type(id) != orgpeg.NodeType_Node {
			return true
		}

		span := tree.Span(id)

		switch tree.Name(id) {
		case "HeadingContent":
			extractTitleAndTags(tree, id, content, &heading)
			return false
		case "EOL":
			if headingLineEOL == 0 {
				headingLineEOL = span.End.Cursor
			}
			return false
		case "Planning":
			planningEnd = span.End.Cursor
			return false
		case "Drawer":
			hasDrawer = true
			drawerStart = span.Start.Cursor
			drawerEnd = span.End.Cursor
			extractDrawerProperties(tree, id, content, &heading)
			return false
		case "Body":
			heading.BodyStart = span.Start.Cursor
			heading.BodyEnd = span.End.Cursor
			return false
		}

		return true
	})

	if hasDrawer {
		drawerBytes := content[drawerStart:drawerEnd]
		if endIdx := bytes.LastIndex(drawerBytes, []byte(":END:")); endIdx >= 0 {
			heading.DrawerEndInsertPos = drawerStart + endIdx
		}
	} else {
		if planningEnd > 0 {
			heading.InsertNewDrawerAt = planningEnd
		} else {
			heading.InsertNewDrawerAt = headingLineEOL
		}
	}

	return heading
}

func extractTitleAndTags(
	tree orgpeg.Tree,
	hcID orgpeg.NodeID,
	content []byte,
	heading *ParsedHeading,
) {
	tree.Visit(hcID, func(id orgpeg.NodeID) bool {
		if id == hcID {
			return true
		}
		if tree.Type(id) != orgpeg.NodeType_Node {
			return true
		}
		span := tree.Span(id)
		switch tree.Name(id) {
		case "HeadingText":
			heading.Title = strings.TrimRight(
				string(content[span.Start.Cursor:span.End.Cursor]),
				" \t",
			)
			return false
		case "Tag":
			heading.Tags = append(
				heading.Tags,
				string(content[span.Start.Cursor:span.End.Cursor]),
			)
			return false
		}
		return true
	})
}

func extractDrawerProperties(
	tree orgpeg.Tree,
	drawerID orgpeg.NodeID,
	content []byte,
	heading *ParsedHeading,
) {
	tree.Visit(drawerID, func(id orgpeg.NodeID) bool {
		if id == drawerID {
			return true
		}
		if tree.Type(id) != orgpeg.NodeType_Node {
			return true
		}
		if tree.Name(id) != "Property" {
			return true
		}

		var key string
		tree.Visit(id, func(pid orgpeg.NodeID) bool {
			if pid == id {
				return true
			}
			if tree.Type(pid) == orgpeg.NodeType_Node && tree.Name(pid) == "PropKey" {
				span := tree.Span(pid)
				key = string(content[span.Start.Cursor:span.End.Cursor])
				return false
			}
			return true
		})

		// Re-read the property line and extract the value (everything
		// after the second ':' on the line, trimmed).
		span := tree.Span(id)
		line := string(content[span.Start.Cursor:span.End.Cursor])

		var value string
		if first := strings.Index(line, ":"); first == 0 {
			if second := strings.Index(line[1:], ":"); second >= 0 {
				value = strings.TrimRight(
					strings.TrimSpace(line[2+second:]),
					"\r\n",
				)
			}
		}

		switch key {
		case "ID":
			heading.ID = value
		case "DODDER_ID":
			heading.DodderID = value
		}

		return false
	})
}

// normalizeIDs walks the headings and generates :ID: values for any that
// lack one. Returns the new content (with IDs spliced in) plus a boolean
// indicating whether any changes were made.
//
// The original content slice is not mutated.
func normalizeIDs(
	content []byte,
	headings []ParsedHeading,
) (newContent []byte, changed bool, err error) {
	type splice struct {
		pos  int
		text string
	}

	var splices []splice
	for _, heading := range headings {
		if heading.ID != "" {
			continue
		}

		generated, uerr := uuid.NewV7()
		if uerr != nil {
			return nil, false, fmt.Errorf("generate uuid v7: %w", uerr)
		}

		if heading.DrawerEndInsertPos > 0 {
			splices = append(splices, splice{
				pos:  heading.DrawerEndInsertPos,
				text: fmt.Sprintf(":ID:       %s\n", generated.String()),
			})
		} else if heading.InsertNewDrawerAt > 0 {
			splices = append(splices, splice{
				pos: heading.InsertNewDrawerAt,
				text: fmt.Sprintf(
					":PROPERTIES:\n:ID:       %s\n:END:\n",
					generated.String(),
				),
			})
		}
	}

	if len(splices) == 0 {
		return content, false, nil
	}

	sort.Slice(splices, func(i, j int) bool {
		return splices[i].pos < splices[j].pos
	})

	var buf bytes.Buffer
	buf.Grow(len(content) + 64*len(splices))
	previous := 0
	for _, s := range splices {
		buf.Write(content[previous:s.pos])
		buf.WriteString(s.text)
		previous = s.pos
	}
	buf.Write(content[previous:])

	return buf.Bytes(), true, nil
}
