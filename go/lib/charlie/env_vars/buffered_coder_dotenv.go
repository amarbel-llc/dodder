package env_vars

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type BufferedCoderDotenv struct{}

func (coder *BufferedCoderDotenv) DecodeFrom(
	envVars EnvVars,
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	clear(envVars)

	eof := false

	for !eof {
		var line string
		line, err = bufferedReader.ReadString('\n')
		n += int64(len(line))

		if err == io.EOF {
			eof = true
			err = nil
		} else if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		line = strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		left, right, ok := strings.Cut(line, "=")

		if !ok {
			err = errors.ErrorWithStackf("malformed env var entry: %q", line)
			return n, err
		}

		envVars[left] = right
	}

	return n, err
}

func (coder BufferedCoderDotenv) EncodeTo(
	envVars EnvVars,
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	var sorted []string

	for key := range envVars {
		sorted = append(sorted, key)
	}

	sort.Strings(sorted)

	var n1 int

	for _, key := range sorted {
		n1, err = fmt.Fprintf(bufferedWriter, "%s=%s\n", key, envVars[key])
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
