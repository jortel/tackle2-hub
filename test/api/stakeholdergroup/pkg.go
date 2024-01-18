package stakeholdergroup

import (
	"github.com/konveyor/tackle2-hub/binding"
	"github.com/konveyor/tackle2-hub/test/api/client"
)

var (
	RichClient       *binding.RichClient
	StakeholderGroup binding.StakeholderGroup
)

func init() {
	// Prepare RichClient and login to Hub API (configured from env variables).
	RichClient = client.PrepareRichClient()

	// Shortcut for StakeholderGroup-related RichClient methods.
	StakeholderGroup = RichClient.StakeholderGroup
}
