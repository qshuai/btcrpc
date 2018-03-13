package main

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

func main() {
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:18332",
		User:         "rpc",
		Pass:         "g6eJ6ZLCiws7mc0SVwlWhI7h4ve5yH",
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown()

	// get listunspent
	lu, err := client.ListUnspent()
	if err != nil {
		panic(err)
	}

	msg := wire.NewMsgTx(1)
	msg.TxIn = make([]*wire.TxIn, 1)
	msg.TxOut = make([]*wire.TxOut, 1)

	for i := 0; i < len(lu); i++ {
		msg.LockTime = 0
		hash, _ := chainhash.NewHashFromStr(lu[i].TxID)
		txin := wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Hash:  *hash,
				Index: 0,
			},
			Sequence: 0xffffff,
		}

		var pkScript []byte
		if i >= len(lu)-1 {
			pkScript, _ = hex.DecodeString(lu[0].ScriptPubKey)
		} else {
			pkScript, _ = hex.DecodeString(lu[i+1].ScriptPubKey)
		}
		txout := wire.TxOut{
			Value:    int64(lu[i].Amount * 1e9 * 0.9),
			PkScript: pkScript,
		}

		msg.TxIn[0] = &txin
		msg.TxOut[0] = &txout
	}

	signtx, _, _ := client.SignRawTransaction(msg)

	ret := client.SendRawTransactionAsync(signtx, true)
	fmt.Println(ret.Receive())
}
