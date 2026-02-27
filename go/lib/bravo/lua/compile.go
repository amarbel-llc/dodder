package lua

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	lua "github.com/yuin/gopher-lua"
	lua_ast "github.com/yuin/gopher-lua/ast"
	lua_parse "github.com/yuin/gopher-lua/parse"
)

func CompileReader(reader io.Reader) (compiled *FunctionProto, err error) {
	var chunks []lua_ast.Stmt

	if chunks, err = lua_parse.Parse(reader, ""); err != nil {
		err = errors.Wrap(err)
		return compiled, err
	}

	if compiled, err = lua.Compile(chunks, ""); err != nil {
		err = errors.Wrap(err)
		return compiled, err
	}

	return compiled, err
}
