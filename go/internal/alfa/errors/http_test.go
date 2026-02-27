package errors

import "testing"

type testErrDisamb struct{}

func TestBadRequestf(t *testing.T) {
	badRequest := BadRequestf("testing")
	badRequest = Wrap(badRequest)

	if !Is400BadRequest(badRequest) {
		t.Errorf("expected bad request")
	}
}

func TestBadRequest(t *testing.T) {
	badRequest := BadRequest(NewWithType[testErrDisamb]("testing"))
	badRequest = Wrap(badRequest)

	if !Is400BadRequest(badRequest) {
		t.Errorf("expected bad request")
	}
}
