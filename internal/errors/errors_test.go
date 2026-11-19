package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestAsKaminoFindsWrappedDomainErrors(t *testing.T) {
	domain := &PathError{KaminoError: KaminoError{Msg: "ruta fuera del home"}}
	wrapped := fmt.Errorf("contexto: %w", domain)
	found, ok := AsKamino(wrapped)
	if !ok {
		t.Fatal("a wrapped domain error must be recognised")
	}
	if found.Error() != "ruta fuera del home" {
		t.Fatalf("unexpected error: %q", found.Error())
	}
	if _, ok := AsKamino(errors.New("error plano")); ok {
		t.Fatal("a plain error is not a domain error")
	}
	if _, ok := AsKamino(nil); ok {
		t.Fatal("nil is not a domain error")
	}
	if _, ok := AsKamino(domain); !ok {
		t.Fatal("a direct domain error must be recognised")
	}
}
