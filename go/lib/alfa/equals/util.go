package equals

func SetIfNotNil[TYPE any](ptr *TYPE, value TYPE) *TYPE {
	if ptr != nil {
		*ptr = value
	}

	return ptr
}

func SetIfValueNotNil[TYPE any](ptr *TYPE, value *TYPE) *TYPE {
	if ptr != nil && value != nil {
		*ptr = *value
	}

	return ptr
}
