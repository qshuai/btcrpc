package main

import (
	"github.com/astaxie/beego/config"
	"github.com/astaxie/beego/logs"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"fmt"
)

const (
	DefaultDust = 1000

	InputLimit = 50
	OutputLimit = 50

	DefaultFee = 0

	DefaultListunspentLimit = 10000
)

// global variables
var (
	log  *logs.BeeLogger
	conf config.Configer

	// store available input and output
	input  = make(coin)
	output = make(map[string][]byte)
	// lessCoin represents transaction with less spendable amount
	lessCoin = make(coin)
	fee int64

	client *rpcclient.Client
	// successful transaction
	count = 0

	s2s *wire.MsgTx
	s2m *wire.MsgTx
	m2s *wire.MsgTx
)

type ref struct {
	hash  chainhash.Hash
	index uint32
}

type coin map[ref]float64

func init() {
	fmt.Println("app init start...")
	// configuration setting
	var err error
	conf, err = config.NewConfig("ini", "conf/app.conf")
	if err != nil {
		panic(err)
	}

	log = logs.NewLogger()
	// log setting
	log.SetLogger("console")
	//log.SetLogger(logs.AdapterFile, `{"filename":"log/btcrpc.log"}`)
	//if must(conf.Bool("log::async")).(bool) {
	//	log.Async(1e3)
	//}

	// get transaction fee from configuration
	fee, err = conf.Int64("tx::fee")
	if err != nil {
		fee = DefaultFee
	}

	client = Client()

	// object reuse
	s2s = wire.NewMsgTx(1)
	s2s.TxIn = make([]*wire.TxIn, 1)
	s2s.TxOut = make([]*wire.TxOut, 1)

	s2m = wire.NewMsgTx(1)
	s2m.TxIn = make([]*wire.TxIn, 1)
	s2m.TxOut = make([]*wire.TxOut, 0, OutputLimit)

	m2s = wire.NewMsgTx(1)
	m2s.TxIn = make([]*wire.TxIn, InputLimit)
	m2s.TxOut = make([]*wire.TxOut, 1)

	fmt.Println("app init complete...")
}

func main() {
	defer client.Shutdown()

	//rangeAccount(client)
	inputs(client)

	dispatch()
}

func signAndSendTx(msg *wire.MsgTx, refs []ref, outs int, recursion bool) {
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
		removeInputRecursion(refs)
		log.Error(err.Error())
	} else {
		removeInputRecursion(refs)

		// recursion tx
		if recursion {
			reference := ref{}
			reference.hash = *txhash
			for i := 0; i < outs; i++ {
				reference.index = uint32(i)
				input[reference] = float64(msg.TxOut[i].Value) * 1e-8
			}
		}
		count++
		log.Info("Create a transaction success, NO.%d, txhash: %s", count, txhash.String())
	}
}

// map return random item
func getRandScriptPubKey() []byte {
	for _, item := range output {
		return item
	}
	return nil
}

func removeInputRecursion(refs []ref) {
	for _, item := range refs{
		delete(input, item)
	}
}

// todo spent to different addresses, support addresses with known ScriptPubKey
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
