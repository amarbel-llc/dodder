package tommy_util

import (
	"io"
	"strconv"

	"github.com/amarbel-llc/tommy/pkg/cst"
	"github.com/amarbel-llc/tommy/pkg/document"
)

func DocumentToMap(doc *document.Document) map[string]any {
	return containerToMap(doc, doc.Root())
}

func ReaderToMap(r io.Reader) (map[string]any, error) {
	doc, err := document.ParseReader(r)
	if err != nil {
		return nil, err
	}

	return DocumentToMap(doc), nil
}

func BytesToMap(b []byte) (map[string]any, error) {
	doc, err := document.Parse(b)
	if err != nil {
		return nil, err
	}

	return DocumentToMap(doc), nil
}

func containerToMap(doc *document.Document, node *cst.Node) map[string]any {
	result := make(map[string]any)

	for _, child := range node.Children {
		switch child.Kind {
		case cst.NodeKeyValue:
			key, value := extractKeyValue(doc, node, child)
			if key != "" {
				result[key] = value
			}

		case cst.NodeTable:
			tableName := extractTableName(child)
			result[tableName] = containerToMap(doc, child)

		case cst.NodeArrayTable:
			tableName := extractTableName(child)
			existing, ok := result[tableName]
			var arr []any
			if ok {
				arr, _ = existing.([]any)
			}

			arr = append(arr, containerToMap(doc, child))
			result[tableName] = arr
		}
	}

	return result
}

func extractKeyValue(
	doc *document.Document,
	container *cst.Node,
	kvNode *cst.Node,
) (string, any) {
	var key string

	for _, child := range kvNode.Children {
		switch child.Kind {
		case cst.NodeKey:
			key = unquoteKey(string(child.Raw))

		case cst.NodeDottedKey:
			key = unquoteKey(string(child.Children[0].Raw))

		case cst.NodeString:
			return key, unquoteString(string(child.Raw))

		case cst.NodeInteger:
			return key, parseInteger(string(child.Raw))

		case cst.NodeFloat:
			return key, parseFloat(string(child.Raw))

		case cst.NodeBool:
			return key, string(child.Raw) == "true"

		case cst.NodeDateTime:
			return key, string(child.Raw)

		case cst.NodeArray:
			return key, extractArray(child)

		case cst.NodeInlineTable:
			return key, containerToMap(doc, child)
		}
	}

	return key, nil
}

func extractArray(node *cst.Node) []any {
	var result []any

	for _, child := range node.Children {
		switch child.Kind {
		case cst.NodeString:
			result = append(result, unquoteString(string(child.Raw)))

		case cst.NodeInteger:
			result = append(result, parseInteger(string(child.Raw)))

		case cst.NodeFloat:
			result = append(result, parseFloat(string(child.Raw)))

		case cst.NodeBool:
			result = append(result, string(child.Raw) == "true")

		case cst.NodeDateTime:
			result = append(result, string(child.Raw))

		case cst.NodeArray:
			result = append(result, extractArray(child))

		case cst.NodeInlineTable:
			result = append(result, containerToMap(nil, child))
		}
	}

	return result
}

func extractTableName(node *cst.Node) string {
	for _, child := range node.Children {
		if child.Kind == cst.NodeKey {
			return unquoteKey(string(child.Raw))
		}

		if child.Kind == cst.NodeDottedKey {
			return unquoteKey(string(child.Children[0].Raw))
		}
	}

	return ""
}

func unquoteKey(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}

	return s
}

func unquoteString(s string) string {
	if len(s) >= 6 && s[:3] == `"""` && s[len(s)-3:] == `"""` {
		return s[3 : len(s)-3]
	}

	if len(s) >= 6 && s[:3] == "'''" && s[len(s)-3:] == "'''" {
		return s[3 : len(s)-3]
	}

	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}

	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}

	return s
}

func parseInteger(s string) int64 {
	v, _ := strconv.ParseInt(s, 0, 64)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
