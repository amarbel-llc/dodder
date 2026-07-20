package domain_interfaces

import (
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
)

type ObjectIOFactory interface {
	ObjectReaderFactory
	ObjectWriterFactory
}

type ObjectReaderFactory interface {
	ObjectReader(mad_domain_interfaces.MarklIdGetter) (mad_domain_interfaces.BlobReader, error)
}

type ObjectWriterFactory interface {
	ObjectWriter() (mad_domain_interfaces.BlobWriter, error)
}

type (
	FuncObjectReader func(mad_domain_interfaces.MarklIdGetter) (mad_domain_interfaces.BlobReader, error)
	FuncObjectWriter func() (mad_domain_interfaces.BlobWriter, error)
)

type bespokeObjectReadWriterFactory struct {
	ObjectReaderFactory
	ObjectWriterFactory
}

func MakeBespokeObjectReadWriterFactory(
	r ObjectReaderFactory,
	w ObjectWriterFactory,
) ObjectIOFactory {
	return bespokeObjectReadWriterFactory{
		ObjectReaderFactory: r,
		ObjectWriterFactory: w,
	}
}

type bespokeObjectReadFactory struct {
	FuncObjectReader
}

func MakeBespokeObjectReadFactory(
	r FuncObjectReader,
) ObjectReaderFactory {
	return bespokeObjectReadFactory{
		FuncObjectReader: r,
	}
}

func (b bespokeObjectReadFactory) ObjectReader(
	sh mad_domain_interfaces.MarklIdGetter,
) (mad_domain_interfaces.BlobReader, error) {
	return b.FuncObjectReader(sh)
}

type bespokeObjectWriterFactory struct {
	FuncObjectWriter
}

func MakeBespokeObjectWriteFactory(
	r FuncObjectWriter,
) ObjectWriterFactory {
	return bespokeObjectWriterFactory{
		FuncObjectWriter: r,
	}
}

func (b bespokeObjectWriterFactory) ObjectWriter() (mad_domain_interfaces.BlobWriter, error) {
	return b.FuncObjectWriter()
}
