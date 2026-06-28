package failure_test

import (
	"errors"
	"testing"

	"github.com/mira4sol/tx-pilot/internal/failure"
	"github.com/mira4sol/tx-pilot/pkg/txpilot"
)

func TestClassifyExpiredBlockhash(t *testing.T) {
	class := failure.Classify(errors.New("blockhash not found"), failure.Evidence{})
	if class.Kind != txpilot.FailureExpiredBlockhash {
		t.Fatalf("got %s", class.Kind)
	}
}

func TestClassifyBundleRejected(t *testing.T) {
	class := failure.Classify(errors.New("bundle rejected by jito"), failure.Evidence{})
	if class.Kind != txpilot.FailureBundleRejected {
		t.Fatalf("got %s", class.Kind)
	}
}
