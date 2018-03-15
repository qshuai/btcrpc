package main

import (
	"github.com/astaxie/beego/config"
	"github.com/astaxie/beego/logs"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcd/txscript"
	"encoding/hex"
	"github.com/btcsuite/btcd/wire"
)

// global variable for log
var log = logs.NewLogger()

// store avaiable input and output
var input = make(map[ref]float64)
var output = make(map[string][]byte)

type ref struct {
	hash  chainhash.Hash
	index uint32
}

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
	msg.TxOut = make([]*wire.TxOut, 10)

	// only support P2PKH transaction
	for i := 0; i < 10000; i++ {
		for reference, amount := range input {
			// skip if the balance of this bitcoin address is zero
			//if amount >  {
			//	continue
			//}

			// construct a P2PKH transaction
			msg.LockTime = 0

			// txin
			txin := wire.TxIn{
				PreviousOutPoint: wire.OutPoint{
					Hash:  reference.hash,
					Index: reference.index,
				},
				Sequence: 0xffffff,
			}

			// txout
			pkScript := getRandScriptPubKey()
			if pkScript == nil {
				panic("no account in output...")
			}

			out := wire.TxOut{
				Value:    int64(amount*1e7 - 1000),
				PkScript: pkScript,
			}

			outNum := 1
			for i := 0; i < outNum; i ++ {
				msg.TxOut[i] = &out
			}
			msg.TxIn[0] = &txin

			// rpc requests signing a raw transaction and gets returned signed transaction,
			// or get null and a err reason
			signed, _, err := client.SignRawTransaction(msg)
			if err != nil {
				log.Error(err.Error())
			}

			// rpc request send a signed transaction, it will return a error if there are any
			// error
			txhash, err := client.SendRawTransaction(signed, true)
			if err != nil {
				delete(input, reference)
				log.Error(err.Error())
			} else {
				delete(input, reference)

				r := ref{}
				for i := 0; i < outNum; i++ {
					r.hash = *txhash
					r.index = uint32(i)
					input[r] = float64(out.Value) * 1e-8
				}
				log.Info("Create a transaction success, txhash: %s", txhash.String())
			}
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
		if item.Amount > 10e-4 && item.Vout < 255 {
			hash, _ := chainhash.NewHashFromStr(item.TxID)
			r := ref{
				hash:  *hash,
				index: item.Vout,
			}
			input[r] = item.Amount

			scriptPubKey, _ := hex.DecodeString(item.ScriptPubKey)
			if err != nil {
				panic(err)
			}
			output[item.Address] = scriptPubKey
		}

		log.Info("input: %d, output: %d", len(input), len(output))
	}
}
