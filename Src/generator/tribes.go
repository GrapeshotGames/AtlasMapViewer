package generator

import (
	"container/heap"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

type TribeCount struct {
	tribeID uint64
	name    string
	count   int
	flagURL *string
	islands []*IslandClaim
}

// TribeCountHeap heap wrapper
type TribeCountHeap []*TribeCount

func (h TribeCountHeap) Len() int { return len(h) }
func (h TribeCountHeap) Less(i, j int) bool {
	if h[i].count < h[j].count {
		return true
	} else if h[i].count == h[j].count {
		return h[i].tribeID < h[j].tribeID
	} else {
		return false
	}
}
func (h TribeCountHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *TribeCountHeap) Push(x interface{}) { *h = append(*h, x.(*TribeCount)) }
func (h *TribeCountHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func TopNTribes(n int, counts *map[uint64]*TribeCount) []uint64 {
	pq := make(TribeCountHeap, 0)
	i := 0
	for _, v := range *counts {
		// prime heap with n-items
		if i < n {
			pq = append(pq, v)
			i++
			continue
		} else if i == n {
			heap.Init(&pq)
		}

		// add n+1 items and fix up heap
		heap.Push(&pq, v)
		if pq.Len() > n {
			heap.Pop(&pq)
		}
		i++
	}
	if i <= n {
		heap.Init(&pq)
	}
	results := make([]uint64, 0)
	for ; pq.Len() > 0; heap.Pop(&pq) {
		results = append(results, pq[0].tribeID)
	}
	for i := len(results)/2 - 1; i >= 0; i-- {
		opp := len(results) - 1 - i
		results[i], results[opp] = results[opp], results[i]
	}
	return results
}

type TribeInfoOutput struct {
	Rank  int    `json:"rank"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Img   string `json:"img"`
}

type TribeOutput struct {
	Version int64                      `json:"version"`
	Top     []string                   `json:"top"`
	Info    map[string]TribeInfoOutput `json:"info"`
}

type GameTribeOutput struct {
	TribeID   uint64 `json:"tribeID"`
	TribeName string `json:"tribeName"`
	Index     int    `json:"index"`
}

func serverIDfromXY(x int, y int) int {
	return (x << 16) | y
}

func generateTribes(client *redis.Client, top []uint64, tribes *map[uint64]*TribeCount, wwwDir string, clusterPrefix string, serversX int, serversY int, wg *sync.WaitGroup) {
	defer wg.Done()

	// fill list of top tribes
	tribeOutput := TribeOutput{
		Version: time.Now().Unix(),
		Top:     make([]string, 0),
		Info:    make(map[string]TribeInfoOutput),
	}

	// start PNG generation and wait a little bit
	for _, tribe := range top {
		randomX := rand.Intn(serversX)
		randomY := rand.Intn(serversY)
		serverID := serverIDfromXY(randomX, randomY)
		client.Publish("GeneralNotifications:GlobalCommands", "Server::"+strconv.Itoa(serverID)+"::GenerateTribePNG "+strconv.FormatUint(tribe, 10))
	}
	time.Sleep(15 * time.Second)

	// fill in tribe info
	var gameTribeOutput []string
	for i := range top {
		strTribeID := strconv.FormatUint(top[i], 10)

		tribeOutput.Top = append(tribeOutput.Top, strTribeID)

		tribeName := "<unknown>"
		img := clusterPrefix + "tribes/na.png"
		tribe, err := client.HMGet("tribedata:"+strTribeID, "TribeName").Result()
		if err != nil {
			log.Println(err)
			continue
		}
		var ok bool
		tribeName, ok = tribe[0].(string)
		if !ok {
			tribeName = "<abandoned>"
		}
		tribeOutput.Info[strTribeID] = TribeInfoOutput{
			Rank:  i + 1,
			Name:  tribeName,
			Color: colors[i],
			Img:   img,
		}

		if tribe, found := (*tribes)[top[i]]; found {
			tribe.name = tribeName
		}

		game := GameTribeOutput{
			TribeID:   top[i],
			TribeName: tribeName,
			Index:     i,
		}
		js, _ := json.Marshal(game)
		gameTribeOutput = append(gameTribeOutput, string(js))
	}

	// try to get the images
	var simpleTribeOutput []TribeInfoOutput
	for i := range top {
		strTribeID := strconv.FormatUint(top[i], 10)
		img, err := client.Get("tribeflag:" + strTribeID).Result()
		if err != nil || len(img) <= 0 {
			simpleTribeOutput = append(simpleTribeOutput, tribeOutput.Info[strTribeID])
			continue
		}
		tribePath := path.Join(wwwDir, clusterPrefix, "tribes", strTribeID+".png")
		os.MkdirAll(path.Dir(tribePath), os.ModePerm)
		ioutil.WriteFile(tribePath, []byte(img), 0644)

		info := tribeOutput.Info[strTribeID]
		info.Img = clusterPrefix + "tribes/" + strTribeID + ".png"
		tribeOutput.Info[strTribeID] = info

		// add flag URL to main tribe data
		if tribe, found := (*tribes)[top[i]]; found {
			tmp := info.Img
			tribe.flagURL = &tmp
		}

		simpleTribeOutput = append(simpleTribeOutput, info)
	}

	// write the json
	js, _ := json.Marshal(tribeOutput)
	tribePath := path.Join(wwwDir, clusterPrefix, "tribes", "tribes.json")
	os.MkdirAll(path.Dir(tribePath), os.ModePerm)
	ioutil.WriteFile(tribePath, []byte(js), 0644)

	// write list back to redis for game
	_, err := client.Del("toptribes").Result()
	if err != nil {
		log.Println(err)
	}
	if len(gameTribeOutput) > 0 {
		_, err = client.RPush("toptribes", gameTribeOutput).Result()
		if err != nil {
			log.Println(err)
		}
	}
	client.Publish("GeneralNotifications:GlobalCommands", "ReloadTopTribes")
}
