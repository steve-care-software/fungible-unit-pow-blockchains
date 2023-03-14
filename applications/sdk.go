package applications

import (
	blockchains "github.com/steve-care-software/blockchains/applications"
	genesis "github.com/steve-care-software/fungible-unit-genesis/applications"
	units "github.com/steve-care-software/fungible-units/applications"
	identities_application "github.com/steve-care-software/identities/applications"
	identities "github.com/steve-care-software/identities/domain"
)

// Application represents the blockchain application
type Application interface {
	Identity() identities_application.Application
	Blockchain() blockchains.Application
	Genesis() genesis.Application
	Units() units.Application
	Authenticate(identity identities.Identity) (Authenticate, error)
}

// Authenticate represents an authenticated application
type Authenticate interface {
}
