package markl

type IdBroken Id

func (id IdBroken) MarshalText() (bites []byte, err error) {
	return bites, err
}

func (id *IdBroken) UnmarshalText(bites []byte) (err error) {
	return err
}
