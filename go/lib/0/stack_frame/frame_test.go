package stack_frame

import (
	"errors" // stdlib: can't use alfa/errors due to import cycle (alfa/errors depends on stack_frame)
	"testing"
)

func TestErrorWrap(t1 *testing.T) {
	var frame Frame
	var ok bool

	if frame, ok = MakeFrame(1); !ok {
		t1.Fatalf("failed to get stack info")
	}

	target := errors.New("sentinel target")

	{
		err := frame.Wrap(target)

		if !errors.Is(err, target) {
			t1.Errorf(
				"expected errors.Is(%#v, %#v) to return true",
				err,
				target,
			)
		}
	}

	{
		err := frame.Wrapf(target, "more info: %s", "hi")

		if !errors.Is(err, target) {
			t1.Errorf(
				"expected errors.Is(%#v, %#v) to return true",
				err,
				target,
			)
		}
	}

	{
		err := frame.Errorf("more info: %s", "hi")

		if errors.Is(err, target) {
			t1.Errorf(
				"expected errors.Is(%#v, %#v) to return false",
				err,
				target,
			)
		}
	}
}

func TestWrapSkip(t *testing.T) {
	t.Skip()
	err := errors.New("sentinal")
	err = WrapSkip(0, err)

	expected := "# TestWrapSkip\nmain_test.go:7: top level"

	if err.Error() != expected {
		t.Errorf("expected %q but got %q", expected, err)
	}
}
