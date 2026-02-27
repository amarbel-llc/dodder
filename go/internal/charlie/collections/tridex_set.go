package collections

// type TridexSet[T schnittstellen.ValueLike] interface {
// 	schnittstellen.SetLike[T]
// 	schnittstellen.TridexLike
// }

// type MutableTridexSet[T schnittstellen.ValueLike] interface {
// 	schnittstellen.MutableSetLike[T]
// 	schnittstellen.TridexLike
// 	GetSet() schnittstellen.SetLike[T]
// }

// func RegisterGobTridexSet[T schnittstellen.ValueLike]() {
// 	gob.Register(&mutableTridexSet[T]{})
// }

// type mutableTridexSet[T schnittstellen.ValueLike] struct {
// 	MS schnittstellen.MutableSetLike[T]
// 	TR schnittstellen.MutableTridex
// }

// func MakeMutableTridexSet[T schnittstellen.ValueLike](
// 	es ...T,
// ) MutableTridexSet[T] {
// 	ms := MakeMutableSetStringer[T](es...)

// 	return mutableTridexSet[T]{
// 		MS: ms,
// 		TR: tridex.Make(iter.Strings[T](ms)...),
// 	}
// }

// func (ms mutableTridexSet[T]) Abbreviate(short string) (long string) {
// 	return ms.TR.Abbreviate(short)
// }

// func (ms mutableTridexSet[T]) Add(e T) (err error) {
// 	if err = ms.MS.Add(e); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	ms.TR.Add(e.String())

// 	return
// }

// func (ms mutableTridexSet[T]) Any() T {
// 	return ms.MS.Any()
// }

// func (ms mutableTridexSet[T]) Contains(e T) bool {
// 	return ms.MS.Contains(e)
// }

// func (ms mutableTridexSet[T]) ContainsKey(e string) bool {
// 	return ms.MS.ContainsKey(e)
// }

// func (ms mutableTridexSet[T]) ContainsAbbreviation(v string) bool {
// 	return ms.TR.ContainsAbbreviation(v)
// }

// func (ms mutableTridexSet[T]) ContainsExpansion(v string) bool {
// 	return ms.TR.ContainsExpansion(v)
// }

// func (ms mutableTridexSet[T]) Del(e T) (err error) {
// 	if err = ms.MS.Del(e); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	ms.TR.Remove(e.String())

// 	return
// }

// func (ms mutableTridexSet[T]) DelKey(v string) (err error) {
// 	if err = ms.MS.DelKey(v); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	ms.TR.Remove(v)

// 	return
// }

// func (ms mutableTridexSet[T]) EachString(
// 	f schnittstellen.FuncIter[string],
// ) (err error) {
// 	return ms.TR.EachString(f)
// }

// func (ms mutableTridexSet[T]) Each(f schnittstellen.FuncIter[T]) (err error)
// {
// 	return ms.MS.Each(f)
// }

// func (ms mutableTridexSet[T]) EachKey(
// 	f schnittstellen.FuncIterKey,
// ) (err error) {
// 	return ms.MS.EachKey(f)
// }

// func (ms mutableTridexSet[T]) Elements() []T {
// 	return ms.MS.Elements()
// }

// func (ms mutableTridexSet[T]) EqualsSetLike(b schnittstellen.SetLike[T]) bool
// {
// 	return ms.MS.EqualsSetLike(b)
// }

// func (ms mutableTridexSet[T]) Expand(short string) (long string) {
// 	return ms.TR.Expand(short)
// }

// func (ms mutableTridexSet[T]) Get(key string) (T, bool) {
// 	return ms.MS.Get(key)
// }

// func (ms mutableTridexSet[T]) GetSet() schnittstellen.SetLike[T] {
// 	return ms.MS.CloneSetLike()
// }

// func (ms mutableTridexSet[T]) Key(e T) string {
// 	return ms.MS.Key(e)
// }

// func (ms mutableTridexSet[T]) Len() int {
// 	return ms.MS.Len()
// }

// func (ms mutableTridexSet[T]) Reset() {
// 	ms.MS.Reset()
// 	// ms.TR.Reset()
// }

// func (ms mutableTridexSet[T]) CloneSetLike() schnittstellen.SetLike[T] {
// 	return ms
// }

// func (ms mutableTridexSet[T]) CloneMutableSetLike()
// schnittstellen.MutableSetLike[T] {
// 	return ms.CloneMutableSetLike()
// }
