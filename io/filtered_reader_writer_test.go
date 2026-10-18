package io_test

import (
	"errors"
	stdio "io"
	"strings"
	"testing"

	guacio "github.com/hq0101/guacamole-common-go/io"
	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestFilteredGuacamoleReader_DropsUntilInstructionPasses(t *testing.T) {
	base := guacio.NewReaderGuacamoleReader(strings.NewReader("4.drop;4.keep;"))
	reader := guacio.NewFilteredGuacamoleReader(base, func(instr protocol.Instruction) (protocol.Instruction, error) {
		if instr.Opcode == "drop" {
			return protocol.Instruction{}, protocol.ErrDropInstruction
		}
		return instr, nil
	})

	instr, err := reader.ReadInstruction()
	if err != nil {
		t.Fatalf("ReadInstruction() error = %v", err)
	}
	if instr.Opcode != "keep" {
		t.Errorf("Opcode = %q, want keep", instr.Opcode)
	}
}

func TestFilteredGuacamoleReader_ReadReturnsFilteredInstruction(t *testing.T) {
	base := guacio.NewReaderGuacamoleReader(strings.NewReader("8.original;"))
	reader := guacio.NewFilteredGuacamoleReader(base, func(protocol.Instruction) (protocol.Instruction, error) {
		return protocol.NewInstruction("mutated"), nil
	})

	raw, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got := string(raw); got != "7.mutated;" {
		t.Errorf("Read() = %q, want mutated instruction", got)
	}
}

func TestFilteredGuacamoleReader_FilterError(t *testing.T) {
	base := guacio.NewReaderGuacamoleReader(strings.NewReader("3.bad;"))
	want := errors.New("denied")
	reader := guacio.NewFilteredGuacamoleReader(base, func(protocol.Instruction) (protocol.Instruction, error) {
		return protocol.Instruction{}, want
	})

	if _, err := reader.ReadInstruction(); err != want {
		t.Fatalf("ReadInstruction() error = %v, want %v", err, want)
	}
}

func TestFilteredGuacamoleReader_EOF(t *testing.T) {
	base := guacio.NewReaderGuacamoleReader(strings.NewReader(""))
	reader := guacio.NewFilteredGuacamoleReader(base, nil)

	if _, err := reader.ReadInstruction(); err != stdio.EOF {
		t.Fatalf("ReadInstruction() error = %v, want io.EOF", err)
	}
}

func TestFilteredGuacamoleWriter_WriteInstruction(t *testing.T) {
	output := &flushBuffer{}
	base := guacio.NewWriterGuacamoleWriter(output)
	writer := guacio.NewFilteredGuacamoleWriter(base, func(protocol.Instruction) (protocol.Instruction, error) {
		return protocol.NewInstruction("mutated", "ok"), nil
	})

	if err := writer.WriteInstruction(protocol.NewInstruction("original")); err != nil {
		t.Fatalf("WriteInstruction() error = %v", err)
	}
	if got := output.String(); got != "7.mutated,2.ok;" {
		t.Errorf("output = %q, want mutated instruction", got)
	}
}

func TestFilteredGuacamoleWriter_DropsInstruction(t *testing.T) {
	output := &flushBuffer{}
	base := guacio.NewWriterGuacamoleWriter(output)
	writer := guacio.NewFilteredGuacamoleWriter(base, func(protocol.Instruction) (protocol.Instruction, error) {
		return protocol.Instruction{}, protocol.ErrDropInstruction
	})

	if err := writer.WriteInstruction(protocol.NewInstruction("drop")); err != nil {
		t.Fatalf("WriteInstruction() error = %v", err)
	}
	if got := output.String(); got != "" {
		t.Errorf("output = %q, want empty", got)
	}
}

func TestFilteredGuacamoleWriter_WriteRangeParsesAndFiltersInstructions(t *testing.T) {
	output := &flushBuffer{}
	base := guacio.NewWriterGuacamoleWriter(output)
	writer := guacio.NewFilteredGuacamoleWriter(base, func(instr protocol.Instruction) (protocol.Instruction, error) {
		if instr.Opcode == "drop" {
			return protocol.Instruction{}, protocol.ErrDropInstruction
		}
		instr.Args = append(instr.Args, "filtered")
		return instr, nil
	})

	if err := writer.WriteRange([]rune("xx4.keep;4.drop;yy"), 2, len("4.keep;4.drop;")); err != nil {
		t.Fatalf("WriteRange() error = %v", err)
	}
	if got := output.String(); got != "4.keep,8.filtered;" {
		t.Errorf("output = %q, want filtered keep only", got)
	}
}

func TestFilteredGuacamoleWriter_WriteIncompleteInstructionReturnsServerError(t *testing.T) {
	output := &flushBuffer{}
	base := guacio.NewWriterGuacamoleWriter(output)
	writer := guacio.NewFilteredGuacamoleWriter(base, nil)

	err := writer.Write([]rune("4.test"))
	if err == nil {
		t.Fatal("Write() error = nil, want incomplete instruction error")
	}
	protoErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("error type = %T, want *protocol.Error", err)
	}
	if protoErr.Status != protocol.StatusServerError {
		t.Errorf("Status = %v, want StatusServerError", protoErr.Status)
	}
}
