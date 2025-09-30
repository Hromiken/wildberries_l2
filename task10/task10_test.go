package main

import (
	"reflect"
	"testing"
)

func TestSortLinesNumeric(t *testing.T) {
	lines := []string{"10", "2", "1"}
	cfg := Config{Numeric: true}
	got := sortLines(lines, cfg)
	want := []string{"1", "2", "10"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Numeric sort failed: got %v, want %v", got, want)
	}
}

func TestSortLinesReverse(t *testing.T) {
	lines := []string{"a", "c", "b"}
	cfg := Config{Reverse: true}
	got := sortLines(lines, cfg)
	want := []string{"c", "b", "a"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Reverse sort failed: got %v, want %v", got, want)
	}
}

func TestSortLinesUnique(t *testing.T) {
	lines := []string{"a", "a", "b", "b", "c"}
	cfg := Config{Unique: true}
	got := sortLines(lines, cfg)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unique failed: got %v, want %v", got, want)
	}
}

func TestSortLinesMonth(t *testing.T) {
	lines := []string{"Mar", "Jan", "Feb"}
	cfg := Config{Month: true}
	got := sortLines(lines, cfg)
	want := []string{"Jan", "Feb", "Mar"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Month sort failed: got %v, want %v", got, want)
	}
}

func TestSortLinesHuman(t *testing.T) {
	lines := []string{"2K", "1M", "512"}
	cfg := Config{Human: true}
	got := sortLines(lines, cfg)
	want := []string{"512", "2K", "1M"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Human-readable sort failed: got %v, want %v", got, want)
	}
}

func TestSortLinesByColumn(t *testing.T) {
	lines := []string{
		"1\tapple", "2\tbanana", "3\tcherry"}
	cfg := Config{Column: 2}
	got := sortLines([]string{lines[2], lines[0], lines[1]}, cfg)
	want := []string{lines[0], lines[1], lines[2]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Column sort failed: got %v, want %v", got, want)
	}
}
