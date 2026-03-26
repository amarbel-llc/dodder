package string_format_writer

import "code.linenisgreat.com/dodder/go/internal/alfa/fields"

const (
	StringDRArrow         = "↳"
	StringNew             = "new"
	StringSame            = "same"
	StringChanged         = "changed"
	StringDeleted         = "deleted"
	StringUpdated         = "updated"
	StringArchived        = "archived"
	StringInternal        = "internal"
	StringUnchanged       = "unchanged"
	StringUntracked       = "untracked"
	StringConflicted      = "conflicted"
	StringRecognized      = "recognized"
	StringCheckedOut      = "checked out"
	StringBlobMissing     = "blob missing"
	StringWouldDelete     = "would delete"
	StringUnrecognized    = "unrecognized"
	StringFormatDateTime  = "06-01-02 15:04:05"
	StringIndent          = "                 "
	StringIndentWithSpace = "                   "
	LenStringMax          = len(StringIndent) // TODO-P4 use reflection?

	colorReset   = "\u001b[0m"
	colorBlack   = "\u001b[30m"
	colorRed     = "\u001b[31m"
	colorGreen   = "\u001b[32m"
	colorYellow  = "\u001b[33m"
	colorBlue    = "\u001b[34m"
	colorMagenta = "\u001b[35m"
	colorCyan    = "\u001b[36m"
	colorWhite   = "\u001b[37m"
	colorItalic  = "\u001b[3m"
	colorNone    = ""

	ColorTypeNormal   = fields.TypeNormal
	ColorTypeId       = fields.TypeId
	ColorTypeHash     = fields.TypeHash
	ColorTypeError    = fields.TypeError
	ColorTypeType     = fields.TypeType
	ColorTypeUserData = fields.TypeUserData
	ColorTypeHeading  = fields.TypeHeading
)
