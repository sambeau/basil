package evaluator

import (
	"testing"

	"github.com/sambeau/basil/testenv"
)

func TestEmail_Send_Placeholder(t *testing.T) {
	// Verify the SMTP test infrastructure is functional.
	env := testenv.Start(t, testenv.WithSMTP())
	if env.SMTPAddr == "" {
		t.Fatal("SMTP server did not start")
	}
	t.Skip("FEAT-084 (email notification API) not yet implemented — enable this test when basil.email.send() lands")
}
