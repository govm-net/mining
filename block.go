package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/lengzhao/govm/wallet"
	"golang.org/x/net/websocket"
)

const (
	// HashLen the byte length of Hash
	HashLen = 32
	// AddressLen the byte length of Address
	AddressLen = 24
)

// Hash The KEY of the block of transaction
type Hash [HashLen]byte

// Address the wallet address
type Address [AddressLen]byte

// Empty Check whether Hash is empty
func (h Hash) Empty() bool {
	return h == (Hash{})
}

// MarshalJSON marshal by base64
func (h Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h[:])
}

// UnmarshalJSON UnmarshalJSON
func (h *Hash) UnmarshalJSON(b []byte) error {
	var v []byte
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	copy(h[:], v)
	return nil
}

// Empty Check where Address is empty
func (a Address) Empty() bool {
	return a == (Address{})
}

// MarshalJSON marshal by base64
func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a[:])
}

// UnmarshalJSON UnmarshalJSON
func (a *Address) UnmarshalJSON(b []byte) error {
	var v []byte
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	copy(a[:], v)
	return nil
}

// Block Block structure
type Block struct {
	//signLen	  uint8
	//sign	      []byte
	Time          uint64
	Previous      Hash
	Parent        Hash
	LeftChild     Hash
	RightChild    Hash
	TransListHash Hash
	Producer      Address
	Chain         uint64
	Index         uint64
	Nonce         uint64
}

// RespBlock Block
type RespBlock struct {
	Block
	HashpowerLimit uint64
	From           string
}

var blocks map[uint64]*RespBlock
var mu sync.Mutex
var hashPowerItem map[int64]uint64
var genBlockNum uint64
var blockFlag int

func init() {
	blocks = make(map[uint64]*RespBlock)
	rand.Seed(time.Now().UnixNano())
	hashPowerItem = make(map[int64]uint64)
}

func showHashPower() {
	now := time.Now().Unix() / 60
	var hp uint64
	var count uint64
	mu.Lock()
	for i := now - 120; i <= now; i++ {
		hp += hashPowerItem[i]
		if hp > 0 {
			count++
		}
	}
	mu.Unlock()
	if count > 0 {
		fmt.Printf("hashpower:%d, generated candidate blocks:%d\n", hp/count, genBlockNum)
	} else {
		fmt.Printf("hashpower:0, generated candidate blocks:%d\n", genBlockNum)
	}
	for _, c := range conf.Chains {
		val := getDataFromServer(c, conf.Servers[0], "", "statMining", wal.AddressStr)
		var count uint64
		if len(val) > 0 {
			Decode(val, &count)
		}
		fmt.Printf("chain:%d, successful mining blocks:%d\n", c, count)
	}
}

type wsHead struct {
	Addr Address
	Time int64
}

func requestBlock(chain uint64, servers chan string) {
	server := <-servers
	defer func(s string) {
		err := recover()
		if err != nil {
			log.Println("recover:request block,", err)
		}
		servers <- s
		time.Sleep(time.Second * 5)
		// log.Printf("chain:%d,disconnect, server:%s\n", chain, server)
		go requestBlock(chain, servers)
	}(server)
	origin := fmt.Sprintf("http://%s", server)
	url := fmt.Sprintf("ws://%s/api/v1/%d/ws/mining", server, chain)
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Println("fail to connect server.", server, err)
		return
	}
	defer ws.Close()
	head := wsHead{}
	Decode(wal.Address, &head.Addr)
	head.Time = time.Now().Unix()
	data := Encode(head)
	sign := wallet.Sign(wal.Key, data)
	data = append(data, sign...)
	_, err = ws.Write(data)
	if err != nil {
		log.Println("send msg error:", err)
		return
	}
	fmt.Printf("chain:%d,connected to the server:%s\n", chain, server)

	for {
		t := time.Now().Add(time.Minute * 2)
		ws.SetReadDeadline(t)
		var block RespBlock
		err = websocket.JSON.Receive(ws, &block)
		if err != nil {
			break
		}
		block.From = server
		mu.Lock()
		blocks[block.Chain] = &block
		blockFlag++
		mu.Unlock()
	}
}

func postBlock(chain uint64, server string, key, data []byte) {
	urlStr := fmt.Sprintf("http://%s/api/v1/%d/data?key=%x&broadcast=true", server, chain, key)
	req, err := http.NewRequest(http.MethodPost, urlStr, bytes.NewBuffer(data))
	if err != nil {
		log.Println("fail to new request:", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("fail to do request:", err)
		return
	}
	// log.Println("response:", resp.Status)
	resp.Body.Close()
}

func updateBlock() {
	for _, c := range conf.Chains {
		servers := make(chan string, len(conf.Servers))
		for _, server := range conf.Servers {
			servers <- server
		}
		for i := 0; i < conf.KeepConnServerNum; i++ {
			go requestBlock(c, servers)
		}
	}
}

func doMining() {
	for _, chain := range conf.Chains {
		for i := 0; i < conf.ThreadNumber; i++ {
			go func(c uint64) {
				for {
					mu.Lock()
					block := blocks[c]
					mu.Unlock()
					if block == nil {
						time.Sleep(20 * time.Second)
						continue
					}
					miner(block)
				}
			}(chain)
		}
	}
}

func miner(in *RespBlock) {
	start := time.Now().Unix()
	if in.Time+80 < uint64(start) {
		time.Sleep(10 * time.Second)
		return
	}

	var block = *in
	block.Nonce = rand.Uint64()
	Decode(wal.Address, &block.Producer)
	// log.Println("start mining")
	var count uint64
	myFlag := blockFlag

	for {
		now := time.Now().Unix()
		if now > start+20 || myFlag != blockFlag {
			id := now / 60
			mu.Lock()
			hashPowerItem[id] += count
			mu.Unlock()
			break
		}
		count++
		block.Nonce++
		data := Encode(block.Block)
		sign := wallet.Sign(wal.Key, data)

		var val = []byte{wallet.SignLen}
		val = append(val, sign...)
		val = append(val, data...)
		key := wallet.GetHash(val)
		if getHashPower(key) >= block.HashpowerLimit {
			log.Printf("mine one candidate block,chain:%d,key:%x\n", block.Chain, key)
			genBlockNum++
			postBlock(block.Chain, block.From, key, val)
		}
	}
}

func getHashPower(in []byte) uint64 {
	var out uint64
	for _, item := range in {
		out += 8
		if item != 0 {
			for item > 0 {
				out--
				item = item >> 1
			}
			return out
		}
	}
	return out
}
