package directory_layout

func stringSliceJoin(s string, vs []string) []string {
	return append([]string{s}, vs...)
}
