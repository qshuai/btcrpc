package main

import (
	"github.com/astaxie/beego/config"
	"github.com/astaxie/beego/logs"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcd/txscript"
	"encoding/hex"
)

// global variable for log
var log = logs.NewLogger()

// store avaiable input and output
var input = make(map[chainhash.Hash]float64)
var output = make(map[string][]byte)

func main() {
	// log setting
	logs.SetLogger(logs.AdapterFile, `{"filename":"log/btcrpc.log"}`)

	// configuration setting
	conf, _ := config.NewConfig("ini", "conf/app.conf")

	// acquire configure item
	link := conf.String("rpc::url") + ":" + conf.String("rpc::port")
	user := conf.String("rpc::user")
	passwd := conf.String("rpc::passwd")

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

	//rangeAccount(client)
	inputs(client)

	msg := wire.NewMsgTx(1)
	msg.TxIn = make([]*wire.TxIn, 1)
	msg.TxOut = make([]*wire.TxOut, 1)

	// only support P2PKH transaction
	for hash, amount := range input {
		// skip if the balance of this bitcoin address is zero
		if amount < 1e-5 {
			continue
		}

		// construct a P2PKH transaction
		msg.LockTime = 0

		// txin
		txin := wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Hash:  hash,
				Index: 0,
			},
			Sequence: 0xffffff,
		}

		// txout
		pkScript := getRandScriptPubKey()
		if pkScript == nil {
			panic("no account in output...")
		}

		out := wire.TxOut{
			Value:    int64(amount * 1e8 * 0.9),
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

func rangeAccount(client *rpcclient.Client) {
	addresses, err := client.GetAddressesByAccount("")
	if err != nil {
		panic(err)
	}

	for _, item := range addresses {
		ret, _, err := base58.CheckDecode(item.String())
		if err != nil {
			panic(err)
		}

		final, err := txscript.NewScriptBuilder().AddOp(txscript.OP_DUP).AddOp(txscript.OP_HASH160).
			AddData(ret).AddOp(txscript.OP_EQUALVERIFY).AddOp(txscript.OP_CHECKSIG).
			Script()

		if err != nil {
			panic(err)
		}
		output[item.String()] = final
	}
}

// map return random item
func getRandScriptPubKey() []byte {
	for _, item := range output {
		return item
	}
	return nil
}

func inputs(client *rpcclient.Client) {
	// rpc requestss to get unspent coin list
	lu, err := client.ListUnspent()
	if err != nil {
		panic(err)
	}

	for _, item := range lu {
		hash, _ := chainhash.NewHashFromStr(item.TxID)
		input[*hash] = item.Amount

		scriptPubKey, _ := hex.DecodeString(item.ScriptPubKey)
		if err != nil {
			panic(err)
		}
		output[item.Address] = scriptPubKey
	}
}
