package applications

import (
	"errors"
	"fmt"
	"strings"

	chains "github.com/steve-care-software/blockchains/domain"
	"github.com/steve-care-software/blockchains/domain/blocks"
	"github.com/steve-care-software/blockchains/domain/transactions"
	fungible_unit_genesis "github.com/steve-care-software/fungible-unit-genesis/applications"
	units "github.com/steve-care-software/fungible-units/applications"
	identities_application "github.com/steve-care-software/identities/applications"
	identities "github.com/steve-care-software/identities/domain"
	"github.com/steve-care-software/libs/cryptography/hash"
	pow_blockchains_application "github.com/steve-care-software/pow-blockchains/applications"
)

type application struct {
	hashAdapter         hash.Adapter
	powBlockchainApp    pow_blockchains_application.Application
	identityApp         identities_application.Application
	genesisApp          fungible_unit_genesis.Application
	fungibleUnitApp     units.Application
	blockBuilder        blocks.Builder
	transactionsBuilder transactions.Builder
	transactionBuilder  transactions.TransactionBuilder
	identity            identities.Identity
}

func createApplication(
	hashAdapter hash.Adapter,
	powBlockchainApp pow_blockchains_application.Application,
	identityApp identities_application.Application,
	genesisApp fungible_unit_genesis.Application,
	fungibleUnitApp units.Application,
	blockBuilder blocks.Builder,
	transactionsBuilder transactions.Builder,
	transactionBuilder transactions.TransactionBuilder,
) Application {
	return createApplicationInternally(
		hashAdapter,
		powBlockchainApp,
		identityApp,
		genesisApp,
		fungibleUnitApp,
		blockBuilder,
		transactionsBuilder,
		transactionBuilder,
		nil,
	)
}

func createApplicationWithIdentity(
	hashAdapter hash.Adapter,
	powBlockchainApp pow_blockchains_application.Application,
	identityApp identities_application.Application,
	genesisApp fungible_unit_genesis.Application,
	fungibleUnitApp units.Application,
	blockBuilder blocks.Builder,
	transactionsBuilder transactions.Builder,
	transactionBuilder transactions.TransactionBuilder,
	identity identities.Identity,
) Application {
	return createApplicationInternally(
		hashAdapter,
		powBlockchainApp,
		identityApp,
		genesisApp,
		fungibleUnitApp,
		blockBuilder,
		transactionsBuilder,
		transactionBuilder,
		identity,
	)
}

func createApplicationInternally(
	hashAdapter hash.Adapter,
	powBlockchainApp pow_blockchains_application.Application,
	identityApp identities_application.Application,
	genesisApp fungible_unit_genesis.Application,
	fungibleUnitApp units.Application,
	blockBuilder blocks.Builder,
	transactionsBuilder transactions.Builder,
	transactionBuilder transactions.TransactionBuilder,
	identity identities.Identity,
) Application {
	out := application{
		hashAdapter:         hashAdapter,
		powBlockchainApp:    powBlockchainApp,
		identityApp:         identityApp,
		genesisApp:          genesisApp,
		fungibleUnitApp:     fungibleUnitApp,
		blockBuilder:        blockBuilder,
		transactionsBuilder: transactionsBuilder,
		transactionBuilder:  transactionBuilder,
		identity:            identity,
	}

	return &out
}

// Identity returns the identity application
func (app *application) Identity() identities_application.Application {
	return app.identityApp
}

// Blockchain returns the blockchain application
func (app *application) Blockchain() pow_blockchains_application.Application {
	return app.powBlockchainApp
}

// Genesis returns the genesis application
func (app *application) Genesis() fungible_unit_genesis.Application {
	return app.genesisApp
}

// Units returns the units application
func (app *application) Units() units.Application {
	return app.fungibleUnitApp
}

// block mines a block then returns it, is executed by the blockchains.EnterOnCreateBlock event
func (app *application) block(chain chains.Chain, body blocks.Body) (blocks.Block, error) {
	// retrieve the genesis:
	root := chain.Root()
	genesis, err := app.powBlockchainApp.Genesis().Retrieve(root)
	if err != nil {
		return nil, err
	}

	// mine the block:
	hash := body.Hash()
	miningValue := genesis.MiningValue()
	difficulty := genesis.Difficulty()
	proof, err := app.powBlockchainApp.Miner().Execute(hash, miningValue, difficulty)
	if err != nil {
		return nil, err
	}

	// build the block with the proof:
	ins, err := app.blockBuilder.Create().WithBody(body).WithProof(proof).Now()
	if err != nil {
		return nil, err
	}

	// return the block
	return ins, nil
}

// validateBlock mines a block then returns it,  is executed by the blockchains.EnterOnCreateBlock event
func (app *application) validateBlock(chain chains.Chain, block blocks.Block) error {
	// retrieve the genesis:
	root := chain.Root()
	genesis, err := app.powBlockchainApp.Genesis().Retrieve(root)
	if err != nil {
		return err
	}

	// create the hash using the proof:
	pMinedHash, err := app.hashAdapter.FromMultiBytes([][]byte{
		block.Proof().Bytes(),
		block.Body().Hash().Bytes(),
	})

	if err != nil {
		return err
	}

	// fetch the completex difficulty:
	completedDifficulty := 0
	mininingValueStr := fmt.Sprintf("%d", genesis.MiningValue())
	pMinedHashStr := pMinedHash.String()
	for idx := range pMinedHashStr {
		if strings.HasPrefix(pMinedHashStr[idx:], mininingValueStr) {
			completedDifficulty++
			continue
		}

		break
	}

	// make sure the completed difficulty is at least the requested amount:
	currentHead := chain.Head()
	requestedDifficulty, err := app.powBlockchainApp.Chain().CalculateNextDifficulty(currentHead)
	if err != nil {
		return err
	}

	if completedDifficulty < int(requestedDifficulty) {
		str := fmt.Sprintf("the completed difficulty (%d) must be greater than the chain's requested difficulty (%d)", completedDifficulty, requestedDifficulty)
		return errors.New(str)
	}

	return nil
}

// signTransaction signs a transaction,  is executed by the blockchains blockchains.EnterOnCreateTransaction
func (app *application) signTransaction(body transactions.Body) (transactions.Transaction, error) {
	if app.identity == nil {
		return nil, errors.New("the identity must be provided in order to sign a transaction")
	}

	hash := body.Hash()
	signature, err := app.identityApp.Sign(hash, app.identity)
	if err != nil {
		return nil, err
	}

	return app.transactionBuilder.Create().
		WithBody(body).
		WithSignature(signature).
		Now()
}

// validateTransaction validates a transaction, is executed by the blockchains blockchains.ExitOnCreateTransaction
func (app *application) validateTransaction(trx transactions.Transaction) error {
	hash := trx.Body().Hash()
	signature := trx.Signature()
	isValid, err := app.identityApp.VerifySignature(hash, signature)
	if err != nil {
		return err
	}

	if !isValid {
		str := fmt.Sprintf("the transaction (hash: %s) contains an invalid signature", trx.Hash().String())
		return errors.New(str)
	}

	return nil
}
