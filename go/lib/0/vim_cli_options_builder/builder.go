package vim_cli_options_builder

import "fmt"

type Builder []string

func New() Builder {
	return Builder(make([]string, 0))
}

func (in Builder) WithFileType(ft string) (out Builder) {
	out = append(in, "-c", fmt.Sprintf("set ft=%s", ft))

	return out
}

func (in Builder) WithSourcedFile(p string) (out Builder) {
	out = append(in, "-c", fmt.Sprintf("source %s", p))

	return out
}

func (in Builder) WithCursorLocation(row, col int) (out Builder) {
	out = append(in, "-c", fmt.Sprintf("call cursor(%d, %d)", row, col))

	return out
}

func (in Builder) WithInsertMode() (out Builder) {
	out = append(in, "-c", "startinsert!")

	return out
}

func (b Builder) Build() []string {
	return []string(b)
}
