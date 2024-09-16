package ledger

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ava-labs/coreth/core/types"
	"github.com/rs/zerolog/log"
)

func signTx(msg *bus.Message) (*types.Transaction, error) {
	w := cmn.CurrentWallet
	if w == nil {
		return nil, errors.New("no wallet")
	}

	m, _ := msg.Data.(*bus.B_SignerSignTx)

	l := provide_device(m.Name)
	if l == nil {
		return nil, fmt.Errorf("Ledger not found: %s", m.Name)
	}

	dp, err := accounts.ParseDerivationPath(m.Path)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error parsing path: %s", m.Path)
		return nil, err
	}

	b := w.GetBlockchain(m.Chain)
	if b == nil {
		return nil, fmt.Errorf("blockchain not found: %v", m.Chain)
	}

	// TODO
	var signedTx *types.Transaction
	dp = dp

	return signedTx, nil
}
