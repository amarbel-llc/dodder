package script_config

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestToml(t1 *testing.T) {
	t := ui.MakeT(t1)

	strToml := `
description = "wow"
file-extension = "pdf"
uti = "com.adobe.pdf"
script = """
cat
"""
  `

	doc, err := DecodeWithOutputFormat([]byte(strToml))
	t.AssertNoError(err)

	sut := doc.Data()

	t.AssertEqualStrings("wow", sut.Description)
	t.AssertEqualStrings("pdf", sut.FileExtension)
	t.AssertEqualStrings("com.adobe.pdf", sut.UTI)
	t.AssertEqualStrings("cat\n", sut.Script)
}
