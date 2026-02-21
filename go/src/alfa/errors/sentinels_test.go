package errors

import (
	"errors"
	"testing"
)

// Test MakeTypedSentinel helper
func TestMakeTypedSentinel(t *testing.T) {
	type testDisamb struct{}

	sentinel, check := MakeTypedSentinel[testDisamb]("test error")

	// Test sentinel is not nil
	if sentinel == nil {
		t.Fatal("MakeTypedSentinel returned nil sentinel")
	}

	// Test error message
	if sentinel.Error() != "test error" {
		t.Errorf("Expected 'test error', got %q", sentinel.Error())
	}

	// Test checker function
	if !check(sentinel) {
		t.Error("Checker function should match sentinel")
	}

	// Test with errors.Is
	if !errors.Is(sentinel, sentinel) {
		t.Error("errors.Is should match sentinel to itself")
	}

	// Test IsTyped works
	if !IsTyped[testDisamb](sentinel) {
		t.Error("IsTyped should match sentinel")
	}

	// Test wrapped sentinel
	wrapped := Wrap(sentinel)
	if !check(wrapped) {
		t.Error("Checker function should work on wrapped errors")
	}

	if !IsTyped[testDisamb](wrapped) {
		t.Error("IsTyped should work on wrapped errors")
	}

	// Test different type doesn't match
	type otherDisamb struct{}
	otherSentinel, _ := MakeTypedSentinel[otherDisamb]("other error")

	if check(otherSentinel) {
		t.Error("Checker function should not match different sentinel type")
	}

	if IsTyped[testDisamb](otherSentinel) {
		t.Error("IsTyped should not match different type")
	}
}

// Test errStopIteration
func TestStopIteration(t *testing.T) {
	err := MakeErrStopIteration()

	// Test sentinel identity
	if !IsStopIteration(err) {
		t.Error("IsStopIteration should match sentinel")
	}

	// Test with errors.Is
	if !errors.Is(err, errStopIteration) {
		t.Error("errors.Is should match sentinel")
	}

	// Test IsTyped works
	if !IsTyped[stopIterationDisamb](err) {
		t.Error("IsTyped should match stop iteration")
	}

	// Test wrapped sentinel
	wrapped := Wrapf(err, "context")
	if !IsStopIteration(wrapped) {
		t.Error("IsStopIteration should work on wrapped errors")
	}

	// Test error message
	if err.Error() != "stop iteration" {
		t.Errorf("Expected 'stop iteration', got %q", err.Error())
	}
}

// Test ErrNotFound
func TestErrNotFound(t *testing.T) {
	err := MakeErrNotFoundString("test-id")

	if !IsErrNotFound(err) {
		t.Error("IsErrNotFound should match ErrNotFound")
	}

	if !IsTyped[errNotFoundDisamb](err) {
		t.Error("IsTyped should match ErrNotFound")
	}

	expected := `not found: "test-id"`
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}

	// Test GetErrNotFound
	notFoundErr, ok := GetErrNotFound(err)
	if !ok {
		t.Fatal("GetErrNotFound should extract ErrNotFound")
	}
	if notFoundErr.Value != "test-id" {
		t.Errorf("Expected Value='test-id', got %q", notFoundErr.Value)
	}

	// Test wrapped error
	wrapped := Wrapf(err, "looking up user")
	if !IsErrNotFound(wrapped) {
		t.Error("IsErrNotFound should work on wrapped errors")
	}

	notFoundErr, ok = GetErrNotFound(wrapped)
	if !ok || notFoundErr.Value != "test-id" {
		t.Error("GetErrNotFound should work on wrapped errors")
	}

	// Test empty value
	emptyErr := MakeErrNotFoundString("")
	if emptyErr.Error() != "not found" {
		t.Errorf("Expected 'not found', got %q", emptyErr.Error())
	}
}

// Test that different error types don't cross-match
func TestTypeSafety(t *testing.T) {
	stopErr := MakeErrStopIteration()
	notFoundErr := MakeErrNotFoundString("test")

	if IsErrNotFound(stopErr) {
		t.Error("Stop iteration should not match not found")
	}

	if IsStopIteration(notFoundErr) {
		t.Error("Not found should not match stop iteration")
	}

	wrappedStop := Wrapf(stopErr, "context")
	wrappedNotFound := Wrapf(notFoundErr, "context")

	if IsErrNotFound(wrappedStop) {
		t.Error("Wrapped stop iteration should not match not found")
	}

	if IsStopIteration(wrappedNotFound) {
		t.Error("Wrapped not found should not match stop iteration")
	}
}

// Test ErrExists
func TestErrExists(t *testing.T) {
	if ErrExists == nil {
		t.Fatal("ErrExists should not be nil")
	}

	if ErrExists.Error() != "exists" {
		t.Errorf("Expected 'exists', got %q", ErrExists.Error())
	}

	if !errors.Is(ErrExists, ErrExists) {
		t.Error("errors.Is should match ErrExists")
	}
}
