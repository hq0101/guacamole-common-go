package io_test

import (
	stdio "io"
	"strings"
	"testing"

	guacio "github.com/hq0101/guacamole-common-go/io"
	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestReaderGuacamoleReader_ReadInstruction(t *testing.T) {
	reader := guacio.NewReaderGuacamoleReader(strings.NewReader("3.nop;4.sync,1.0;"))

	first, err := reader.ReadInstruction()
	if err != nil {
		t.Fatalf("first ReadInstruction() error = %v", err)
	}
	if first.Opcode != "nop" || len(first.Args) != 0 {
		t.Fatalf("first instruction = %+v, want nop", first)
	}

	second, err := reader.ReadInstruction()
	if err != nil {
		t.Fatalf("second ReadInstruction() error = %v", err)
	}
	if second.Opcode != "sync" || len(second.Args) != 1 || second.Args[0] != "0" {
		t.Fatalf("second instruction = %+v, want sync [0]", second)
	}

	if _, err := reader.ReadInstruction(); err != stdio.EOF {
		t.Fatalf("third ReadInstruction() error = %v, want io.EOF", err)
	}
}

func TestReaderGuacamoleReader_ReadReturnsRawChars(t *testing.T) {
	reader := guacio.NewReaderGuacamoleReader(strings.NewReader("000003.nop;"))

	raw, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got := string(raw); got != "000003.nop;" {
		t.Errorf("Read() = %q, want raw instruction", got)
	}
}

func TestWriterGuacamoleWriter_WriteInstructionFlushes(t *testing.T) {
	output := &flushBuffer{}
	writer := guacio.NewWriterGuacamoleWriter(output)

	if err := writer.WriteInstruction(protocol.NewInstruction("sync", "0")); err != nil {
		t.Fatalf("WriteInstruction() error = %v", err)
	}
	if got := output.String(); got != "4.sync,1.0;" {
		t.Errorf("output = %q, want sync instruction", got)
	}
	if output.flushes != 1 {
		t.Errorf("flushes = %d, want 1", output.flushes)
	}
}

func TestWriterGuacamoleWriter_WriteRange(t *testing.T) {
	output := &flushBuffer{}
	writer := guacio.NewWriterGuacamoleWriter(output)

	if err := writer.WriteRange([]rune("xx3.nop;yy"), 2, len("3.nop;")); err != nil {
		t.Fatalf("WriteRange() error = %v", err)
	}
	if got := output.String(); got != "3.nop;" {
		t.Errorf("output = %q, want nop instruction", got)
	}
	if output.flushes != 1 {
		t.Errorf("flushes = %d, want 1", output.flushes)
	}
}

type flushBuffer struct {
	strings.Builder
	flushes int
}

func (b *flushBuffer) Flush() error {
	b.flushes++
	return nil
}
