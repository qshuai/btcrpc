package main

import (
	"encoding/hex"

	"github.com/astaxie/beego/config"
	"github.com/astaxie/beego/logs"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

// global variable for log
var log = logs.NewLogger()

func main() {
	// log setting
	logs.SetLogger(logs.AdapterFile, `{"filename":"log/btcrpc.log"}`)

	// configuration setting
	conf, _ := config.NewConfig("ini", "conf/app.conf")

	// acquire configure item
	link := conf.String("rpc:url") + conf.String("rpc:url")
	user := conf.String("rpc:user")
	passwd := conf.String("rpc:passwd")

	// rpc client instance
	connCfg := &rpcclient.ConnConfig{
		Host:         link,
		User:         user,
		Pass:         passwd,
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

	// rpc requestss to get unspent coin list
	lu, err := client.ListUnspent()
	if err != nil {
		panic(err)
	}

	msg := wire.NewMsgTx(1)
	msg.TxIn = make([]*wire.TxIn, 1)
	msg.TxOut = make([]*wire.TxOut, 1)

	// only support P2PKH transaction
	for i := 0; i < len(lu); i++ {
		// construct a P2PKH transaction
		msg.LockTime = 0
		hash, _ := chainhash.NewHashFromStr(lu[i].TxID)
		// txin
		txin := wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Hash:  *hash,
				Index: 0,
			},
			Sequence: 0xffffff,
		}

		// txout
		var pkScript []byte
		if i >= len(lu)-1 {
			pkScript, _ = hex.DecodeString(lu[0].ScriptPubKey)
		} else {
			pkScript, _ = hex.DecodeString(lu[i+1].ScriptPubKey)
		}
		out := wire.TxOut{
			Value:    int64(lu[i].Amount * 1e9 * 0.9),
			PkScript: pkScript,
		}

		msg.TxIn[0] = &txin
		msg.TxOut[0] = &out

		// rpc requests signing a raw transaction and gets returned signed transaction,
		// or get null and a err reason
		signed, _, err := client.SignRawTransaction(msg)
		if err != nil {
			log.Error(err.Error())
		}

		// rpc request send a signed transaction, it will return a error if there are any
		// error
		ret := client.SendRawTransactionAsync(signed, true)
		if txhash, err := ret.Receive(); err != nil {
			log.Error(err.Error())
		} else {
			log.Info("Create a transaction success, txhash: %s", txhash.String())
		}
	}
}
