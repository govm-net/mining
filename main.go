package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

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

const version = "v0.5.0"

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

	var cmd string
	var descList = []string{
		"nil",
		"show HashPower",
		"show block for mining",
		"show wallet address",
		"show private key of wallet",
		"enter private key of wallet",
		"quit",
	}
	for {
		ops, _ := strconv.ParseInt(cmd, 10, 32)
		switch ops {
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
			fmt.Printf("wallet: %x\n", wal.Address)
		case 4:
			fmt.Printf("Private key: %x\n", wal.Key)
		case 5:
			var keyStr string
			fmt.Println("please entry private key:")
			fmt.Scanln(&keyStr)
			privKey, err := hex.DecodeString(keyStr)
			if err != nil || len(privKey) != 32 {
				fmt.Println("error Private key,", err, len(privKey))
				break
			}
			pubKey := wallet.GetPublicKey(privKey)
			address := wallet.PublicKeyToAddress(pubKey, wallet.EAddrTypeDefault)
			fmt.Printf("the wallet address:%x\nentry 'yes' to save wallet.", address)
			var save string
			fmt.Scanln(&save)
			if strings.ToLower(save) == "yes" {
				fmt.Println("save and resplace wallet")
				wal.Key = privKey
				wal.Address = address
				os.Rename(conf.WalletFile, "old_"+conf.WalletFile)
				wallet.SaveWallet(conf.WalletFile, conf.Password, wal.Address, wal.Key, nil)
			} else {
				fmt.Println("do not save")
			}
		case 6:
			fmt.Println("exiting")
			time.Sleep(time.Second)
			os.Exit(0)
		default:
			fmt.Println("Please enter the operation number")
			for i, it := range descList {
				if i == 0 {
					continue
				}
				fmt.Printf("  %d: %s\n", i, it)
			}
		}
		cmd = ""
		fmt.Scanln(&cmd)
	}
}
