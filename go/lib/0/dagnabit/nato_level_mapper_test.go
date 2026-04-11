package dagnabit

import "testing"

func TestNATOLevelMapperHeight0(t *testing.T) {
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "0" {
		t.Errorf("expected %q, got %q", "0", name)
	}
}

func TestNATOLevelMapperHeight1(t *testing.T) {
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "alfa" {
		t.Errorf("expected %q, got %q", "alfa", name)
	}
}

func TestNATOLevelMapperMaxHeight(t *testing.T) {
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(26)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "zulu" {
		t.Errorf("expected %q, got %q", "zulu", name)
	}
}

func TestNATOLevelMapperOutOfRange(t *testing.T) {
	m := MakeNATOLevelMapper()

	_, err := m.LevelName(27)
	if err == nil {
		t.Fatal("expected error for out-of-range height, got nil")
	}
}
