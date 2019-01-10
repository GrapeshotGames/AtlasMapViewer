package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

// EntityInfo record in redis.
type EntityInfo struct {
	EntityID                string
	ParentEntityID          string
	EntityType              string
	EntitySubType           string
	EntityName              string
	TribeID                 string
	ServerXRelativeLocation float64
	ServerYRelativeLocation float64
	ServerID                [2]uint16
	LastUpdatedDBAt         uint64
	NextAllowedUseTime      uint64
}

// Configuration options for the server.
type Configuration struct {
	Host               string
	Port               uint16
	TerritoryURL       string
	StaticDir          string
	DisableCommands    bool
	FetchRateInSeconds int
	RedisAddress       string
	RedisPassword      string
	RedisDB            int
}

// ServerLocation relative percentage to a specific server's origin.
type ServerLocation struct {
	ServerID                [2]uint16
	ServerXRelativeLocation float64
	ServerYRelativeLocation float64
}

var gameData map[string]EntityInfo
var gameDataLock sync.RWMutex

var tribeData map[string]string
var tribeDataLock sync.RWMutex

var config Configuration

func main() {
	var err error
	config, err = loadConfig("./config.json")

	gameData = make(map[string]EntityInfo)
	tribeData = make(map[string]string)

	if err != nil {
		log.Printf("Warning: %v", err)
		log.Println("Failed to read configuration file: config.json")
	}

	go fetch()

	http.HandleFunc("/gettribes", getTribes)
	http.HandleFunc("/getdata", getData)
	http.HandleFunc("/command", sendCommand)
	http.HandleFunc("/travels", getPathTravelled)
	http.HandleFunc("/territoryURL", getTerritoryURL)
	http.Handle("/", http.FileServer(http.Dir(config.StaticDir)))

	endpoint := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Println("Listening on ", endpoint)
	log.Fatal(http.ListenAndServe(endpoint, nil))
}

func loadConfig(path string) (cfg Configuration, err error) {
	var file *os.File

	file, err = os.Open(path)
	defer file.Close()
	if err != nil {
		return
	}

	decoder := json.NewDecoder(file)

	cfg = Configuration{
		Port:               8880,
		TerritoryURL:       "http://localhost:8881/territoryTiles/",
		StaticDir:          "./www",
		DisableCommands:    true,
		FetchRateInSeconds: 15,
		RedisAddress:       "localhost:6379",
		RedisPassword:      "foobared",
		RedisDB:            0,
	}

	err = decoder.Decode(&cfg)
	return
}

func fetch() {
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	var kidsWithBadParents map[string]bool
	kidsWithBadParents = make(map[string]bool)

	for {
		tribes := make(map[string]string)
		entities := make(map[string]EntityInfo)

		for _, record := range scan(client, "tribedata:*") {
			tribes[record["TribeID"]] = record["TribeName"]
		}
		tribeDataLock.Lock()
		tribeData = tribes
		tribeDataLock.Unlock()

		for _, record := range scan(client, "entityinfo:*") {
			// log.Println(id)
			info := newEntityInfo(record)
			entities[info.EntityID] = *info
		}

		// sanity check entity data, e.g. any missing parent ids?
		for k, v := range entities {
			if v.ParentEntityID != "0" {
				if _, parentFound := entities[v.ParentEntityID]; !parentFound {
					if _, dontSpamLog := kidsWithBadParents[k]; !dontSpamLog {
						kidsWithBadParents[k] = true
						log.Printf("Entity %s references parent %s that does not exist, removing from list", k, v.ParentEntityID)
					}
					delete(entities, k)
				}
			}
		}

		gameDataLock.Lock()
		gameData = entities
		gameDataLock.Unlock()

		time.Sleep(time.Duration(config.FetchRateInSeconds) * time.Second)
	}
}

func scan(client *redis.Client, pattern string) map[string]map[string]string {
	records := make(map[string]map[string]string)

	start := time.Now()

	// Scan is slower than Keys but provides gaps for other things to execute
	var keys []string
	iter := client.Scan(0, pattern, 5000).Iterator()
	for iter.Next() {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Fatal(err)
	}

	// Load each entity
	results := make(map[string]*redis.StringStringMapCmd)
	pipe := client.Pipeline()
	for _, id := range keys {
		results[id] = pipe.HGetAll(id)
	}
	pipe.Exec()
	for _, id := range keys {
		records[id], _ = results[id].Result()
	}

	elapsed := time.Since(start)
	log.Printf("Redis scan took %s", elapsed)

	return records
}

// serverID unpacks the packed server ID. Each Server has an X and Y ID which
// corresponds to its 2D location in the game world. The ID is packed into
// 32-bits as follows:
//   +--------------+--------------+
//   | X (uint16_t) | Y (uint16_t) |
//   +--------------+--------------+
func serverID(packed string) (split [2]uint16, err error) {
	var id uint64
	id, err = strconv.ParseUint(packed, 10, 32)
	if err != nil {
		return
	}

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(id))
	split[0] = binary.LittleEndian.Uint16(buf[:2])
	split[1] = binary.LittleEndian.Uint16(buf[2:])
	return
}

// newEntityInfo transforms a Key-Value record into a new EntityInfo object.
func newEntityInfo(record map[string]string) *EntityInfo {
	var info EntityInfo
	info.EntityID = record["EntityID"]
	info.ParentEntityID = record["ParentEntityID"]
	info.EntityName = record["EntityName"]
	info.EntityType = record["EntityType"]
	info.ServerXRelativeLocation, _ = strconv.ParseFloat(record["ServerXRelativeLocation"], 64)
	info.ServerYRelativeLocation, _ = strconv.ParseFloat(record["ServerYRelativeLocation"], 64)
	info.LastUpdatedDBAt, _ = strconv.ParseUint(record["LastUpdatedDBAt"], 10, 64)
	info.NextAllowedUseTime, _ = strconv.ParseUint(record["NextAllowedUseTime"], 10, 64)

	var ok bool
	var tmpTribeID string
	if tmpTribeID, ok = record["TribeID"]; !ok {
		tmpTribeID, _ = record["TribeId"]
	}
	info.TribeID = tmpTribeID

	var tmpServerID string
	if tmpServerID, ok = record["ServerID"]; !ok {
		tmpServerID, _ = record["ServerId"]
	}
	info.ServerID, _ = serverID(tmpServerID)

	// convert entity class to a subtype
	var tmpEntityClass string
	if tmpEntityClass, ok = record["EntityClass"]; !ok {
		tmpEntityClass = "none"
	}
	tmpEntityClass = strings.ToLower(tmpEntityClass)
	if strings.Contains(tmpEntityClass, "none") {
		info.EntitySubType = "None"
	} else if strings.Contains(tmpEntityClass, "brigantine") {
		info.EntitySubType = "Brigantine"
	} else if strings.Contains(tmpEntityClass, "dinghy") {
		info.EntitySubType = "Dingy"
	} else if strings.Contains(tmpEntityClass, "raft") {
		info.EntitySubType = "Raft"
	} else if strings.Contains(tmpEntityClass, "sloop") {
		info.EntitySubType = "Sloop"
	} else if strings.Contains(tmpEntityClass, "schooner") {
		info.EntitySubType = "Schooner"
	} else if strings.Contains(tmpEntityClass, "galleon") {
		info.EntitySubType = "Galleon"
	} else {
		info.EntitySubType = "None"
	}

	return &info
}

func getTerritoryURL(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	wrapper := make(map[string]string)
	wrapper["url"] = config.TerritoryURL

	js, err := json.Marshal(wrapper)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

func getTribes(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	tribeDataLock.RLock()
	js, err := json.Marshal(tribeData)
	tribeDataLock.RUnlock()

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

// getData returns the latest game data pulled from the backend.
func getData(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	gameDataLock.RLock()
	js, err := json.Marshal(gameData)
	gameDataLock.RUnlock()

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

// getPathTravelled returns a fake path for the specified ship.
func getPathTravelled(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	shipID := r.URL.Query().Get("id")

	gameDataLock.RLock()
	info, ok := gameData[shipID]
	gameDataLock.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var path []ServerLocation
	path = append(path, ServerLocation{
		info.ServerID,
		info.ServerXRelativeLocation,
		info.ServerYRelativeLocation,
	})

	for i, end := 0, rand.Intn(10); i <= end; i++ {
		path = append(path, ServerLocation{
			info.ServerID,
			path[len(path)-1].ServerXRelativeLocation * rand.Float64(),
			path[len(path)-1].ServerYRelativeLocation * rand.Float64(),
		})
	}

	js, err := json.Marshal(path)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// sendCommand publishes an event to the GeneralNotifications:GlobalCommands
// redis PubSub channel. To send a server command, prepend "ID::X,Y::" where
// ID is the packed server ID; X and Y are the relative lng and lat locations.
func sendCommand(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" || config.DisableCommands {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	//encoded := fmt.Sprintf("Map::%s", body)
	encoded := fmt.Sprintf("%s", body)

	log.Println("publish:", encoded)
	result, err := client.Publish("GeneralNotifications:GlobalCommands", encoded).Result()
	if err != nil {
		log.Println("redis error for: ", encoded, "; ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write([]byte(strconv.FormatInt(result, 10)))
}
