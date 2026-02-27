package string_format_writer

type CliFormatTruncation int

const (
	CliFormatTruncationNone = CliFormatTruncation(iota)
	CliFormatTruncation66CharEllipsis
)
