package transactions_builder

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/data_storage/accounts"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/registrations"
	"pandora-pay/blockchain/data_storage/tokens"
	"pandora-pay/blockchain/data_storage/tokens/token"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/cryptography/bn256"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	advanced_connection_types "pandora-pay/network/websocks/connection/advanced-connection-types"
	"pandora-pay/store"
	store_db_interface "pandora-pay/store/store-db/store-db-interface"
	"pandora-pay/transactions-builder/wizard"
)

func (builder *TransactionsBuilder) CreateZetherRing(from, dst string, token []byte, ringSize int, newAccounts int) ([]string, error) {

	var addr *addresses.Address
	var err error

	if ringSize == -1 {
		pow := rand.Intn(4) + 4
		ringSize = int(math.Pow(2, float64(pow)))
	}
	if newAccounts == -1 {
		newAccounts = rand.Intn(ringSize / 5)
	}

	if ringSize < 0 {
		return nil, errors.New("number is negative")
	}
	if !crypto.IsPowerOf2(ringSize) {
		return nil, errors.New("ring size is not a power of 2")
	}
	if newAccounts < 0 || newAccounts > ringSize-2 {
		return nil, errors.New("New accounts needs to be in the interval [0, ringSize-2] ")
	}

	alreadyUsed := make(map[string]bool)
	if addr, err = addresses.DecodeAddr(from); err != nil {
		return nil, err
	}
	alreadyUsed[string(addr.PublicKey)] = true

	if addr, err = addresses.DecodeAddr(dst); err != nil {
		return nil, err
	}
	alreadyUsed[string(addr.PublicKey)] = true

	rings := make([]string, ringSize-2)

	if err := store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		accsCollection := accounts.NewAccountsCollection(reader)
		regs := registrations.NewRegistrations(reader)

		var accs *accounts.Accounts
		if accs, err = accsCollection.GetMap(token); err != nil {
			return
		}

		for i := 0; i < len(rings); i++ {

			if regs.Count < uint64(ringSize) {
				priv := addresses.GenerateNewPrivateKey()
				if addr, err = priv.GenerateAddress(true, 0, nil); err != nil {
					return
				}
			} else {

				var acc *account.Account
				if acc, err = accs.GetRandomAccount(); err != nil {
					return
				}
				if acc == nil {
					errors.New("Error getting any random account")
				}

				if addr, err = addresses.CreateAddr(acc.PublicKey, nil, 0, nil); err != nil {
					return
				}

			}
			if alreadyUsed[string(addr.PublicKey)] {
				i--
				continue
			}
			alreadyUsed[string(addr.PublicKey)] = true
			rings[i] = addr.EncodeAddr()
		}

		return
	}); err != nil {
		return nil, err
	}

	return rings, nil
}

func (builder *TransactionsBuilder) CreateZetherTx_Float(from []string, tokensUsed [][]byte, amounts []float64, dsts []string, burns []float64, ringMembers [][]string, data []*wizard.TransactionsWizardData, fees []*TransactionsBuilderFeeFloat, propagateTx, awaitAnswer, awaitBroadcast bool, ctx context.Context, statusCallback func(string)) (*transaction.Transaction, error) {

	amountsFinal := make([]uint64, len(amounts))
	burnsFinal := make([]uint64, len(burns))
	finalFees := make([]*wizard.TransactionsWizardFee, len(fees))

	statusCallback("Converting Floats to Numbers")

	if err := store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		toks := tokens.NewTokens(reader)

		for i := range amounts {
			if err != nil {
				return
			}

			var tok *token.Token
			if tok, err = toks.GetToken(tokensUsed[i]); err != nil {
				return
			}
			if tok == nil {
				return errors.New("Token was not found")
			}

			if amountsFinal[i], err = tok.ConvertToUnits(amounts[i]); err != nil {
				return
			}
			if burnsFinal[i], err = tok.ConvertToUnits(burns[i]); err != nil {
				return
			}
			if finalFees[i], err = fees[i].convertToWizardFee(tok); err != nil {
				return
			}
		}

		return
	}); err != nil {
		return nil, err
	}

	return builder.CreateZetherTx(from, tokensUsed, amountsFinal, dsts, burnsFinal, ringMembers, data, finalFees, propagateTx, awaitAnswer, awaitBroadcast, ctx, statusCallback)
}

func (builder *TransactionsBuilder) CreateZetherTx(from []string, tokensUsed [][]byte, amounts []uint64, dsts []string, burns []uint64, ringMembers [][]string, data []*wizard.TransactionsWizardData, fees []*wizard.TransactionsWizardFee, propagateTx, awaitAnswer, awaitBroadcast bool, ctx context.Context, statusCallback func(string)) (*transaction.Transaction, error) {

	if len(from) != len(tokensUsed) || len(tokensUsed) != len(amounts) || len(amounts) != len(dsts) || len(dsts) != len(burns) || len(burns) != len(data) || len(data) != len(fees) {
		return nil, errors.New("Length of from and transfers are not matching")
	}

	fromWalletAddresses, err := builder.getWalletAddresses(from)
	if err != nil {
		return nil, err
	}

	builder.lock.Lock()
	defer builder.lock.Unlock()

	var tx *transaction.Transaction
	var chainHeight uint64
	var chainHash []byte

	transfers := make([]*wizard.ZetherTransfer, len(from))

	emap := make(map[string]map[string][]byte) //initialize all maps
	rings := make([][]*bn256.G1, len(from))

	publicKeyIndexes := make(map[string]*wizard.ZetherPublicKeyIndex)

	if err := store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		accsCollection := accounts.NewAccountsCollection(reader)
		regs := registrations.NewRegistrations(reader)

		chainHeight, _ = binary.Uvarint(reader.Get("chainHeight"))
		chainHash = helpers.CloneBytes(reader.Get("chainHash"))

		for _, token := range tokensUsed {
			if emap[string(token)] == nil {
				emap[string(token)] = map[string][]byte{}
			}
		}

		for i, fromWalletAddress := range fromWalletAddresses {

			var accs *accounts.Accounts
			if accs, err = accsCollection.GetMap(tokensUsed[i]); err != nil {
				return
			}

			transfers[i] = &wizard.ZetherTransfer{
				Token:       tokensUsed[i],
				From:        fromWalletAddress.PrivateKey.Key[:],
				Destination: dsts[i],
				Amount:      amounts[i],
				Burn:        burns[i],
				Data:        data[i],
			}

			var ring []*bn256.G1

			addPoint := func(address string) (err error) {
				var addr *addresses.Address
				var p *crypto.Point
				if addr, err = addresses.DecodeAddr(address); err != nil {
					return
				}
				if p, err = addr.GetPoint(); err != nil {
					return
				}

				var acc *account.Account
				if acc, err = accs.GetAccount(addr.PublicKey); err != nil {
					return
				}

				var balance []byte
				if acc != nil {
					balance = acc.Balance.Amount.Serialize()
				}

				if balance, err = builder.mempool.GetZetherBalance(addr.PublicKey, balance); err != nil {
					return
				}

				if fromWalletAddress.AddressEncoded == address { //sender

					balancePoint := new(crypto.ElGamal)
					if balancePoint, err = balancePoint.Deserialize(balance); err != nil {
						return
					}

					var fromBalanceDecoded uint64
					if fromBalanceDecoded, err = builder.wallet.DecodeBalanceByPublicKey(fromWalletAddress.PublicKey, balancePoint, tokensUsed[i], true, true, ctx, statusCallback); err != nil {
						return
					}

					if fromBalanceDecoded == 0 {
						return errors.New("You have no funds")
					}

					if fromBalanceDecoded < amounts[i] {
						return errors.New("Not enough funds")
					}
					transfers[i].FromBalanceDecoded = fromBalanceDecoded

				}

				emap[string(tokensUsed[i])][p.G1().String()] = balance

				ring = append(ring, p.G1())

				var isReg bool
				if isReg, err = regs.Exists(string(addr.PublicKey)); err != nil {
					return
				}

				publicKeyIndex := &wizard.ZetherPublicKeyIndex{}
				publicKeyIndexes[string(addr.PublicKey)] = publicKeyIndex

				publicKeyIndex.Registered = isReg
				if isReg {
					if publicKeyIndex.RegisteredIndex, err = regs.GetIndexByKey(string(addr.PublicKey)); err != nil {
						return
					}
				} else {
					publicKeyIndex.RegistrationSignature = addr.Registration
				}

				return
			}

			if err = addPoint(fromWalletAddress.AddressEncoded); err != nil {
				return
			}
			if err = addPoint(dsts[i]); err != nil {
				return
			}

			for _, ringMember := range ringMembers[i] {
				if err = addPoint(ringMember); err != nil {
					return
				}
			}

			rings[i] = ring
		}
		statusCallback("Wallet Addresses Found")

		return
	}); err != nil {
		return nil, err
	}
	statusCallback("Balances checked")

	if tx, err = wizard.CreateZetherTx(transfers, emap, rings, chainHeight, chainHash, publicKeyIndexes, fees, ctx, statusCallback); err != nil {
		gui.GUI.Error("Error creating Tx: ", err)
		return nil, err
	}

	statusCallback("Transaction Created")
	if propagateTx {
		if err := builder.mempool.AddTxToMemPool(tx, chainHeight, awaitAnswer, awaitBroadcast, advanced_connection_types.UUID_ALL); err != nil {
			return nil, err
		}
	}

	return tx, nil
}
