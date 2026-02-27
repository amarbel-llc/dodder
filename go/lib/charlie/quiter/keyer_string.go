package quiter

var StringKeyer stringKeyer

type stringKeyer struct{}

func (stringKeyer) GetKey(e string) string {
	return e
}

type stringKeyerPtr struct{}

func (stringKeyerPtr) GetKey(e string) string {
	return e
}

func (stringKeyerPtr) GetKeyPtr(e *string) string {
	if e == nil {
		return ""
	}

	return *e
}
