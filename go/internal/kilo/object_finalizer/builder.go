package object_finalizer

type builder struct {
	verifyOptions VerifyOptions
}

func Builder() *builder {
	return &builder{
		verifyOptions: defaultVerifyOptions,
	}
}

func (builder *builder) WithVerifyOptionObjectSigPresent(present bool) *builder {
	builder.verifyOptions.ObjectSigPresent = present
	return builder
}

func (builder *builder) WithVerifyOptions(verifyOptions VerifyOptions) *builder {
	builder.verifyOptions = verifyOptions
	return builder
}

func (builder *builder) Build() Finalizer {
	return finalizer{
		verifyOptions: builder.verifyOptions,
	}
}
