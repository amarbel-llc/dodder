package alfred

type Mod struct {
	Valid    bool   `json:"valid"`
	Arg      string `json:"arg"`
	Subtitle string `json:"subtitle"`
}

func (i *Mod) Reset() {
	i.Valid = true
	i.Arg = ""
	i.Subtitle = ""
}
