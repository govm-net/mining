package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/lengzhao/govm/wallet"
)

// Config config
type Config struct {
	WalletFile   string   `json:"wallet_file,omitempty"`
	Password     string   `json:"password,omitempty"`
	Servers      []string `json:"servers,omitempty"`
	ThreadNumber int      `json:"thread_number,omitempty"`
	Chains       []uint64 `json:"chains,omitempty"`
}

const version = "v0.1.0"

var conf Config
var wal wallet.TWallet

func loadConfig(fileName string) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Println("fail to read configure.", err)
		os.Exit(2)
	}
	err = json.Unmarshal(data, &conf)
	if err != nil {
		log.Println("fail to Unmarshal configure.", err)
		os.Exit(2)
	}
	if len(conf.Servers) == 0 {
		log.Println("request servers")
		os.Exit(2)
	}
}

// loadWallet load wallet
func loadWallet(fileName, password string) {
	var err error
	wal, err = wallet.LoadWallet(fileName, password)
	if err != nil {
		if _, err = os.Stat(fileName); !os.IsNotExist(err) {
			log.Println("fail to load wallet.", err)
			os.Exit(4)
		}
		wal.Key = wallet.NewPrivateKey()
		pubKey := wallet.GetPublicKey(wal.Key)
		wal.Address = wallet.PublicKeyToAddress(pubKey, wallet.EAddrTypeDefault)
		wallet.SaveWallet(fileName, password, wal.Address, wal.Key, wal.SignPrefix)
	}
	fmt.Printf("wallet: %x\n", wal.Address)
}

func main() {
	fmt.Println("version of govm mining:", version)
	loadConfig("./conf.json")
	loadWallet(conf.WalletFile, conf.Password)

	addr := hex.EncodeToString(wal.Address)
	for _, chain := range conf.Chains {
		var stat bool
		for _, server := range conf.Servers {
			if isMiner(chain, server, addr) {
				stat = true
				break
			}
		}
		if !stat {
			fmt.Println("warning, not miner, chain:", chain)
		} else {
			fmt.Println("enable mining, chain:", chain)
		}
	}
	updateBlock()
	doMining()

	var cmd int
	var descList = []string{
		"nil",
		"show HashPower",
		"show block for mining",
		"show wallet address",
	}
	for {
		switch cmd {
		case 1:
			showHashPower()
		case 2:
			mu.Lock()
			for c, block := range blocks {
				if block == nil {
					continue
				}
				fmt.Printf("chain:%d,index:%d,hp limit:%d,previous:%x\n",
					c, block.Index, block.HashpowerLimit, block.Previous)
			}
			mu.Unlock()
		case 3:
			fmt.Printf("wallet:%x\n", wal.Address)
		default:
			fmt.Println("Please enter the operation number")
			for i, it := range descList {
				if i == 0 {
					continue
				}
				fmt.Printf("  %d: %s\n", i, it)
			}
		}
		cmd = 0
		fmt.Scanln(&cmd)
	}
}
