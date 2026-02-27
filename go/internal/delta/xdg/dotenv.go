package xdg

import (
	"fmt"
	"io"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
)

// TODO replace with env_vars.BufferedCoderDotenv
type Dotenv struct {
	*XDG
}

func (dotenv Dotenv) setDefaultOrEnvFromMap(
	defawlt string,
	envKey string,
	out *string,
	env map[string]string,
) (err error) {
	if value, ok := env[envKey]; ok {
		*out = value
	} else {
		*out = os.Expand(defawlt, func(v string) string {
			switch v {
			case "HOME":
				return dotenv.Home.String()

			default:
				return os.Getenv(v)
			}
		})
	}

	return err
}

func (dotenv Dotenv) ReadFrom(reader io.Reader) (n int64, err error) {
	env := make(map[string]string)

	bufferedReader, repool := pool.GetBufferedReader(reader)
	defer repool()

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

		env[left] = right
	}

	toInitialize := dotenv.getInitElements()

	for _, ie := range toInitialize {
		if err = dotenv.setDefaultOrEnvFromMap(
			ie.defawlt.TemplateDefault,
			ie.defawlt.Name,
			&ie.actual.ActualValue,
			env,
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}

func (dotenv Dotenv) WriteTo(writer io.Writer) (n int64, err error) {
	bufferedWriter, repool := pool.GetBufferedWriter(writer)
	defer repool()

	initElements := dotenv.getInitElements()
	var n1 int

	for _, initElement := range initElements {
		n1, err = fmt.Fprintf(
			bufferedWriter,
			"%s=%s\n",
			initElement.defawlt.Name,
			initElement.actual.ActualValue,
		)
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
