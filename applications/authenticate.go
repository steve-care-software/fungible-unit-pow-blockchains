package applications

import (
	"errors"
	"fmt"
	"strings"

	chains "github.com/steve-care-software/blockchains/domain"
	"github.com/steve-care-software/blockchains/domain/blocks"
	"github.com/steve-care-software/blockchains/domain/transactions"
	identities "github.com/steve-care-software/identities/domain"
	"github.com/steve-care-software/libs/cryptography/hash"
	pow_blockchains_application "github.com/steve-care-software/pow-blockchains/applications"
)

type authenticate struct {
	hashAdapter         hash.Adapter
	powBlockchainApp    pow_blockchains_application.Application
	blockBuilder        blocks.Builder
	transactionsBuilder transactions.Builder
	transactionBuilder  transactions.TransactionBuilder
	identity            identities.Identity
}

func createAuthenticate(
	hashAdapter hash.Adapter,
	powBlockchainApp pow_blockchains_application.Application,
	blockBuilder blocks.Builder,
	transactionsBuilder transactions.Builder,
	transactionBuilder transactions.TransactionBuilder,
	identity identities.Identity,
) Authenticate {
	out := authenticate{
		hashAdapter:         hashAdapter,
		powBlockchainApp:    powBlockchainApp,
		blockBuilder:        blockBuilder,
		transactionsBuilder: transactionsBuilder,
		transactionBuilder:  transactionBuilder,
		identity:            identity,
	}

	return &out
}

// Block mines a block then returns it, is executed by the blockchains.EnterOnCreateBlock event
func (app *authenticate) Block(chain chains.Chain, body blocks.Body) (blocks.Block, error) {
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

// ValidateBlock mines a block then returns it,  is executed by the blockchains.EnterOnCreateBlock event
func (app *authenticate) ValidateBlock(chain chains.Chain, block blocks.Block) error {
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

// SignTransaction signs a transaction,  is executed by the blockchains blockchains.EnterOnCreateTransaction
func (app *authenticate) SignTransaction(body transactions.Body) error {
	return nil
}

// ValidateTransaction validates a transaction, is executed by the blockchains blockchains.EnterOnCreateTransaction
func (app *authenticate) ValidateTransaction(trx transactions.Transaction) error {
	return nil
}
