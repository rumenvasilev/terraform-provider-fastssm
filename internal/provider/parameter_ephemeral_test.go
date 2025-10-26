package provider

import (
	"testing"
)

func TestAccParameterEphemeral(t *testing.T) {
	t.Skip("Ephemeral resources cannot be meaningfully tested in isolation. " +
		"They don't save to state (can't use TestCheckResourceAttr) and can only be " +
		"used in provider configuration blocks. To properly test ephemeral functionality, " +
		"we would need to use the ephemeral resource to configure a different provider " +
		"and verify that provider received the correct values. " +
		"The e2e tests in tests/e2e/ provide better coverage for ephemeral resource usage.")
}
