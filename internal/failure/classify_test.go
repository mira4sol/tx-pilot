package failure_test

import (
	"errors"
	"testing"

	"github.com/mira4sol/aegis/internal/failure"
	"github.com/mira4sol/aegis/pkg/aegis"
)

func TestClassifyExpiredBlockhash(t *testing.T) {
	class := failure.Classify(errors.New("blockhash not found"), failure.Evidence{})
	if class.Kind != aegis.FailureExpiredBlockhash {
		t.Fatalf("got %s", class.Kind)
	}
}

func TestClassifyBundleRejected(t *testing.T) {
	class := failure.Classify(errors.New("bundle rejected by jito"), failure.Evidence{})
	if class.Kind != aegis.FailureBundleRejected {
		t.Fatalf("got %s", class.Kind)
	}
}
