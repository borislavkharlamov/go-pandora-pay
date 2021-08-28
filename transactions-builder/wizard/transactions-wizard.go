package wizard

import (
	"errors"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/transactions/transaction"
	transaction_simple "pandora-pay/blockchain/transactions/transaction/transaction-simple"
	"pandora-pay/blockchain/transactions/transaction/transaction-simple/transaction-simple-extra"
	"pandora-pay/blockchain/transactions/transaction/transaction-simple/transaction-simple-parts"
	transaction_type "pandora-pay/blockchain/transactions/transaction/transaction-type"
	"pandora-pay/cryptography"
)

func signSimpleTransaction(tx *transaction.Transaction, privateKey *addresses.PrivateKey, statusCallback func(string)) (err error) {

	statusCallback("Transaction Signing...")

	if tx.TransactionBaseInterface.(*transaction_simple.TransactionSimple).Vin.Signature, err = privateKey.Sign(tx.SerializeForSigning()); err != nil {
		return err
	}

	statusCallback("Transaction Signed")

	return
}

func CreateSimpleTxOneInOneOut(nonce uint64, token []byte, key []byte, amount uint64, dst string, dstAmount uint64, data *TransactionsWizardData, fee *TransactionsWizardFee, statusCallback func(string)) (*transaction.Transaction, error) {
	return CreateSimpleTx(nonce, token, [][]byte{key}, []uint64{amount}, []string{dst}, []uint64{dstAmount}, data, fee, statusCallback)
}

func CreateSimpleTx(nonce uint64, token []byte, keys [][]byte, amounts []uint64, dsts []string, dstsAmounts []uint64, data *TransactionsWizardData, fee *TransactionsWizardFee, statusCallback func(string)) (*transaction.Transaction, error) {
	//
	//if len(keys) != len(amounts) || len(amounts) == 0 {
	//	return nil, errors.New("Input lengths are a mismatch")
	//}
	//if len(dsts) != len(dstsAmounts) {
	//	return nil, errors.New("Output lengths are a mismatch")
	//}
	//
	//privateKeys := make([]*addresses.PrivateKey, len(keys))
	//vin := make([]*transaction_simple_parts.TransactionSimpleInput, len(keys))
	//for i := 0; i < len(keys); i++ {
	//
	//	privateKeys[i] = &addresses.PrivateKey{Key: keys[i]}
	//
	//	vin[i] = &transaction_simple_parts.TransactionSimpleInput{
	//		PublicKey: privateKeys[i].GeneratePublicKey(),
	//	}
	//}
	//
	//vout := make([]*transaction_simple_parts.TransactionSimpleOutput, len(dsts))
	//for i := 0; i < len(dsts); i++ {
	//	outAddress, err := addresses.DecodeAddr(dsts[i])
	//	if err != nil {
	//		return nil, err
	//	}
	//	vout[i] = &transaction_simple_parts.TransactionSimpleOutput{
	//		PublicKey: outAddress.PublicKey,
	//		Amount:    dstsAmounts[i],
	//	}
	//}
	//
	//dataFinal, err := data.getData()
	//if err != nil {
	//	return nil, err
	//}
	//
	//tx := &transaction.Transaction{
	//	Version:     transaction_type.TX_SIMPLE,
	//	DataVersion: data.getDataVersion(),
	//	Data:        dataFinal,
	//	TransactionBaseInterface: &transaction_simple.TransactionSimple{
	//		Nonce: nonce,
	//		Token: token,
	//		Vin:   vin,
	//		Vout:  vout,
	//	},
	//}
	//
	//statusCallback("Transaction created")
	//
	//if err := signSimpleTransaction(tx, privateKeys, statusCallback); err != nil {
	//	return nil, err
	//}
	//
	//if err := setFee(tx, &TransactionsWizardFeeExtra{*fee, false}); err != nil {
	//	return nil, err
	//}
	//statusCallback("Transaction Fees set")
	//
	//if err := signSimpleTransaction(tx, privateKeys, statusCallback); err != nil {
	//	return nil, err
	//}
	//
	//if err := tx.BloomAll(); err != nil {
	//	return nil, err
	//}
	//statusCallback("Transaction Bloomed")
	//
	//if err := tx.Validate(); err != nil {
	//	return nil, err
	//}
	//statusCallback("Transaction Validated")
	//
	//if err := tx.Verify(); err != nil {
	//	return nil, err
	//}
	//statusCallback("Transaction Verified")
	//
	//return tx, nil

	panic("not implemented")
}

func CreateUnstakeTx(nonce uint64, key []byte, unstakeAmount uint64, data *TransactionsWizardData, fee *TransactionsWizardFee, statusCallback func(string)) (*transaction.Transaction, error) {

	privateKey := &addresses.PrivateKey{Key: key}

	dataFinal, err := data.getData()
	if err != nil {
		return nil, err
	}

	tx := &transaction.Transaction{
		Version:     transaction_type.TX_SIMPLE,
		DataVersion: data.getDataVersion(),
		Data:        dataFinal,
		TransactionBaseInterface: &transaction_simple.TransactionSimple{
			TxScript: transaction_simple.SCRIPT_UNSTAKE,
			Nonce:    nonce,
			Fee:      0,
			TransactionSimpleExtraInterface: &transaction_simple_extra.TransactionSimpleUnstake{
				Amount: unstakeAmount,
			},
			Vin: &transaction_simple_parts.TransactionSimpleInput{
				PublicKey: privateKey.GeneratePublicKey(),
			},
		},
	}
	statusCallback("Transaction Created")

	if err := signSimpleTransaction(tx, privateKey, statusCallback); err != nil {
		return nil, err
	}

	if err := setFee(tx, fee); err != nil {
		return nil, err
	}
	statusCallback("Transaction Fees set")

	if err := signSimpleTransaction(tx, privateKey, statusCallback); err != nil {
		return nil, err
	}

	if err := tx.BloomAll(); err != nil {
		return nil, err
	}
	statusCallback("Transaction Bloomed")

	if err := tx.Validate(); err != nil {
		return nil, err
	}
	statusCallback("Transaction Validated")

	if err := tx.Verify(); err != nil {
		return nil, err
	}
	statusCallback("Transaction Verified")

	return tx, nil
}

func CreateUpdateDelegateTx(nonce uint64, key []byte, delegateNewPubKey []byte, delegateNewFee uint16, data *TransactionsWizardData, fee *TransactionsWizardFee, statusCallback func(string)) (*transaction.Transaction, error) {

	dataFinal, err := data.getData()
	if err != nil {
		return nil, err
	}

	if len(delegateNewPubKey) != cryptography.PublicKeySize {
		return nil, errors.New("Delegating arguments are empty")
	}

	privateKey := &addresses.PrivateKey{Key: key}
	tx := &transaction.Transaction{
		Version:     transaction_type.TX_SIMPLE,
		DataVersion: data.getDataVersion(),
		Data:        dataFinal,
		TransactionBaseInterface: &transaction_simple.TransactionSimple{
			TxScript: transaction_simple.SCRIPT_UPDATE_DELEGATE,
			Nonce:    nonce,
			TransactionSimpleExtraInterface: &transaction_simple_extra.TransactionSimpleUpdateDelegate{
				NewPublicKey: delegateNewPubKey,
				NewFee:       delegateNewFee,
			},
			Vin: &transaction_simple_parts.TransactionSimpleInput{
				PublicKey: privateKey.GeneratePublicKey(),
			},
		},
	}
	statusCallback("Transaction Created")

	if err := signSimpleTransaction(tx, privateKey, statusCallback); err != nil {
		return nil, err
	}

	if err := setFee(tx, fee); err != nil {
		return nil, err
	}
	statusCallback("Transaction Fees set")

	if err := signSimpleTransaction(tx, privateKey, statusCallback); err != nil {
		return nil, err
	}

	if err := tx.BloomAll(); err != nil {
		return nil, err
	}
	statusCallback("Transaction Bloomed")

	if err := tx.Validate(); err != nil {
		return nil, err
	}
	statusCallback("Transaction Validated")

	if err := tx.Verify(); err != nil {
		return nil, err
	}
	statusCallback("Transaction Verified")

	return tx, nil
}
