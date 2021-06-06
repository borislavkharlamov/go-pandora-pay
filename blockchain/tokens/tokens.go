package tokens

import (
	"encoding/json"
	"errors"
	"pandora-pay/blockchain/tokens/token"
	token_info "pandora-pay/blockchain/tokens/token-info"
	"pandora-pay/config"
	"pandora-pay/cryptography"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"pandora-pay/store/hash-map"
	store_db_interface "pandora-pay/store/store-db/store-db-interface"
)

type Tokens struct {
	hash_map.HashMap `json:"-"`
}

func NewTokens(tx store_db_interface.StoreDBTransactionInterface) (tokens *Tokens) {
	tokens = &Tokens{
		HashMap: *hash_map.CreateNewHashMap(tx, "Tokens", cryptography.PublicKeyHashHashSize),
	}
	tokens.HashMap.Deserialize = func(data []byte) (helpers.SerializableInterface, error) {
		var tok = &token.Token{}
		err := tok.Deserialize(helpers.NewBufferReader(data))
		return tok, err
	}
	return
}

func (tokens *Tokens) GetToken(key []byte) (tok *token.Token, err error) {

	if len(key) == 0 {
		key = config.NATIVE_TOKEN_FULL
	}

	data, err := tokens.HashMap.Get(string(key))
	if data == nil || err != nil {
		return
	}

	tok = data.(*token.Token)

	return
}

func (tokens *Tokens) CreateToken(key []byte, tok *token.Token) (err error) {

	if len(key) == 0 {
		key = config.NATIVE_TOKEN_FULL
	}

	if err = tok.Validate(); err != nil {
		return
	}

	gui.GUI.Log("WWWWWWWWWWWWWWWWWWwaaaa4_1")

	var exists bool
	if exists, err = tokens.ExistsToken(key); err != nil {
		return
	}
	if exists {
		return errors.New("token already exists")
	}

	gui.GUI.Log("WWWWWWWWWWWWWWWWWWwaaaa4_2")
	tokens.UpdateToken(key, tok)
	return
}

func (tokens *Tokens) UpdateToken(key []byte, tok *token.Token) {

	if len(key) == 0 {
		key = config.NATIVE_TOKEN_FULL
	}

	tokens.Update(string(key), tok)
}

func (tokens *Tokens) ExistsToken(key []byte) (bool, error) {
	if len(key) == 0 {
		key = config.NATIVE_TOKEN_FULL
	}

	return tokens.Exists(string(key))
}

func (tokens *Tokens) DeleteToken(key []byte) {
	tokens.Delete(string(key))
}

func (hashMap *Tokens) WriteToStore() (err error) {

	if err = hashMap.HashMap.WriteToStore(); err != nil {
		return
	}

	if config.SEED_WALLET_NODES_INFO {

		for k, v := range hashMap.Committed {

			if v.Stored == "del" {
				err = hashMap.Tx.DeleteForcefully("tokenInfo_ByHash" + k)
			} else if v.Stored == "update" {

				tok := v.Element.(*token.Token)
				tokInfo := &token_info.TokenInfo{
					Hash:             []byte(k),
					Name:             tok.Name,
					Ticker:           tok.Ticker,
					DecimalSeparator: tok.DecimalSeparator,
					Description:      tok.Description,
				}
				var data []byte
				data, err = json.Marshal(tokInfo)

				err = hashMap.Tx.Put("tokenInfo_ByHash"+k, data)
			}

			if err != nil {
				return
			}
		}

	}

	return
}
