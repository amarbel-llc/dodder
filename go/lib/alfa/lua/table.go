package lua

func ClearTable(s *LState, t *LTable) {
	// ui.Debug().Print(t.Len())
	// defer ui.Debug().Print(t.Len())
	s.ForEach(
		t,
		func(keyValue, _ LValue) {
			key := keyValue.(LString).String()
			t.RawSetString(key, LNil)
		},
	)
}
