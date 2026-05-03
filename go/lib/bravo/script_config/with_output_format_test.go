package script_config

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
)

func TestToml(t1 *testing.T) {
	t := ui.T{T: t1}

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

	if sut.Description != "wow" {
		t.Errorf("expected Description 'wow' but got %q", sut.Description)
	}

	if sut.FileExtension != "pdf" {
		t.Errorf("expected FileExtension 'pdf' but got %q", sut.FileExtension)
	}

	if sut.UTI != "com.adobe.pdf" {
		t.Errorf("expected UTI 'com.adobe.pdf' but got %q", sut.UTI)
	}

	if sut.Script != "cat\n" {
		t.Errorf("expected Script 'cat\\n' but got %q", sut.Script)
	}
}
