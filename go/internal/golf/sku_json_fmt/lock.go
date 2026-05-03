package sku_json_fmt

type Lock struct {
	Type             string            `json:"type,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
	References       map[string]string `json:"references,omitempty"`
	ReferenceAliases map[string]string `json:"reference-aliases,omitempty"`
}

type BlobReference struct {
	TypeLockKey   string `json:"type-lock-key,omitempty"`
	TypeLockValue string `json:"type-lock-value,omitempty"`
	Alias         string `json:"alias,omitempty"`
}
