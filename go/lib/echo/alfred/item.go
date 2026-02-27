package alfred

import (
	"code.linenisgreat.com/dodder/go/lib/delta/catgut"
)

type Item struct {
	Title        string         `json:"title,omitempty"`
	Arg          string         `json:"arg,omitempty"`
	Subtitle     string         `json:"subtitle,omitempty"`
	Match        *catgut.String `json:"match,omitempty"`
	Icon         ItemIcon       `json:"icon,omitempty"`
	Uid          string         `json:"uid,omitempty"`
	ItemType     string         `json:"type,omitempty"`
	QuicklookUrl string         `json:"quicklookurl,omitempty"`
	Text         ItemText       `json:"text,omitempty"`
	Mods         map[string]Mod `json:"mods,omitempty"`
	// Valid        bool     `json:"valid,omitempty"`
}

func (item *Item) Reset() {
	item.Title = ""
	item.Arg = ""
	item.Subtitle = ""
	item.Match.Reset()
	item.Icon.Type = ""
	item.Icon.Path = ""
	item.Uid = ""
	item.QuicklookUrl = ""
	item.Text.Copy = ""
	clear(item.Mods)
}

type ItemText struct {
	Copy string `json:"copy,omitempty"`
}

type ItemIcon struct {
	Type string `json:"type,omitempty"`
	Path string `json:"path,omitempty"`
}
