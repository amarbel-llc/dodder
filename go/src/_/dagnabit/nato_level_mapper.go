package dagnabit

import "fmt"

var natoLevels = []string{
	"_",
	"alfa",
	"bravo",
	"charlie",
	"delta",
	"echo",
	"foxtrot",
	"golf",
	"hotel",
	"india",
	"juliett",
	"kilo",
	"lima",
	"mike",
	"november",
	"oscar",
	"papa",
	"quebec",
	"romeo",
	"sierra",
	"tango",
	"uniform",
	"victor",
	"whiskey",
	"xray",
	"yankee",
	"zulu",
}

type NATOLevelMapper struct{}

func MakeNATOLevelMapper() NATOLevelMapper {
	return NATOLevelMapper{}
}

func (m NATOLevelMapper) LevelName(height int) (string, error) {
	if height < 0 || height >= len(natoLevels) {
		return "", fmt.Errorf(
			"height %d out of range (max %d)",
			height,
			len(natoLevels)-1,
		)
	}

	return natoLevels[height], nil
}
