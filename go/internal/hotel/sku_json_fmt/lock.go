package sku_json_fmt

type Lock struct {
	Type       string            `json:"type,omitempty"`
	References map[string]string `json:"references,omitempty"`
}
