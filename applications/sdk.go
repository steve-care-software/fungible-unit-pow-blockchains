package applications

import (
	fungible_unit_genesis "github.com/steve-care-software/fungible-unit-genesis/applications"
	units "github.com/steve-care-software/fungible-units/applications"
	identities_application "github.com/steve-care-software/identities/applications"
	identities "github.com/steve-care-software/identities/domain"
	pow_blockchains_application "github.com/steve-care-software/pow-blockchains/applications"
)

// Builder represents the application builder
type Builder interface {
	Create() Builder
	WithIdentity(identity identities.Identity) Builder
	Now() (Application, error)
}

// Application represents the blockchain application
type Application interface {
	Identity() identities_application.Application
	Blockchain() pow_blockchains_application.Application
	Genesis() fungible_unit_genesis.Application
	Units() units.Application
}
