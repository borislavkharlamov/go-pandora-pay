package addresses

import (
	"github.com/stretchr/testify/assert"
	"pandora-pay/cryptography"
	"pandora-pay/helpers"
	"testing"
)

func TestGenerateNewPrivateKey(t *testing.T) {

	privateKey := GenerateNewPrivateKey()
	assert.Equal(t, len(privateKey.Key), 32, "Invalid private key length")
	assert.NotEqual(t, privateKey.Key, helpers.EmptyBytes(32), "Invalid private key is empty")

}

func TestPrivateKey_GenerateAddress(t *testing.T) {

	privateKey := GenerateNewPrivateKey()

	address, err := privateKey.GenerateAddress(0, helpers.EmptyBytes(0))
	assert.NoError(t, err)

	assert.Equal(t, len(address.PublicKey), cryptography.PublicKeySize, "Generated Address is invalid")
	assert.NotEqual(t, address.PublicKey, helpers.EmptyBytes(cryptography.PublicKeySize), "Generated Address is invalid")
	assert.Equal(t, address.Amount, uint64(0), "Generated Address is invalid")
	assert.Equal(t, len(address.PaymentID), 0, "Generated Address is invalid")

	address, err = privateKey.GenerateAddress(0, helpers.EmptyBytes(0))
	assert.NoError(t, err)

	assert.Equal(t, len(address.PublicKey), 0, "Generated Address is invalid")
	assert.Equal(t, len(address.PublicKey), cryptography.PublicKeySize, "Generated Address is invalid")
	assert.NotEqual(t, address.PublicKey, helpers.EmptyBytes(cryptography.PublicKeySize), "Generated Address is invalid")
	assert.Equal(t, address.Amount, uint64(0), "Generated Address is invalid")
	assert.Equal(t, len(address.PaymentID), 0, "Generated Address is invalid")

	address, err = privateKey.GenerateAddress(20, helpers.RandomBytes(8))
	assert.NoError(t, err)

	assert.Equal(t, len(address.PublicKey), 0, "Generated Address is invalid")
	assert.Equal(t, len(address.PublicKey), cryptography.PublicKeySize, "Generated Address is invalid")
	assert.NotEqual(t, address.PublicKey, helpers.EmptyBytes(cryptography.PublicKeySize), "Generated Address is invalid")
	assert.Equal(t, address.Amount, uint64(20), "Generated Address is invalid")
	assert.Equal(t, len(address.PaymentID), 8, "Generated Address is invalid")

}
