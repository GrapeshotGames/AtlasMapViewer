package main

import (
	"flag"
	"log"
	"path/filepath"
	"strconv"
	"net/http"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"sync"

	"AtlasMapViewer/atlas"
	"AtlasMapViewer/generator"

	"github.com/go-redis/redis"
)

var islandData string
var islandDataLock sync.RWMutex

var entityData string
var entityDataLock sync.RWMutex

var tribeData string
var tribeDataLock sync.RWMutex


// sendCommand publishes an event to the GeneralNotifications:GlobalCommands
// redis PubSub channel. To send a server command, prepend "ID::X,Y::" where
// ID is the packed server ID; X and Y are the relative lng and lat locations.
func sendCommand(w http.ResponseWriter, r *http.Request, client *redis.Client, config *generator.Config) {
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


func getTerritoryURL(w http.ResponseWriter, r *http.Request, config *generator.Config) {
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

func getIslands(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	islandDataLock.RLock()
	w.Write([]byte(islandData))
	islandDataLock.RUnlock()
}

func getEntities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	entityDataLock.RLock()
	w.Write([]byte(entityData))
	entityDataLock.RUnlock()
}

func getTribes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	tribeDataLock.RLock()
	w.Write([]byte(tribeData))
	tribeDataLock.RUnlock()
}

func main() {
	atlasDirPtr := flag.String("atlas", ".", "Directory containing Atlas ServerGrid.ServerOnly.json and ServerGrid.json files")
	genConfigFilePtr := flag.String("config", "./config.json", "Generator config file")
	flag.Parse()

	serverOnlyConfig, err := atlas.LoadSeverOnlyConfig(filepath.Join(*atlasDirPtr, "ServerGrid.ServerOnly.json"))
	if err != nil {
		log.Fatal(err)
	}
	gridConfig, err := atlas.LoadGridConfig(filepath.Join(*atlasDirPtr, "ServerGrid.json"))
	if err != nil {
		log.Fatal(err)
	}
	generatorConfig, err := generator.LoadConfig(*genConfigFilePtr)
	if err != nil {
		log.Fatal(err)
	}
	dbTribeCfg, _ := serverOnlyConfig.GetDatabaseByName("TribeDB")
	dbTribeClient := redis.NewClient(&redis.Options{
		Addr:     dbTribeCfg.URL + ":" + strconv.Itoa(dbTribeCfg.Port),
		Password: dbTribeCfg.Password,
		DB:       0,
	})

	dbTerritoryCfg, _ := serverOnlyConfig.GetDatabaseByName("TerritoryDB")
	dbTerritoryClient := redis.NewClient(&redis.Options{
		Addr:     dbTerritoryCfg.URL + ":" + strconv.Itoa(dbTerritoryCfg.Port),
		Password: dbTerritoryCfg.Password,
		DB:       0,
	})

	islandData = "{}"
	entityData = "{}"
	tribeData = "{}"
	if generatorConfig.ColonyFetchRateInSeconds > 0 {
		go generator.ProcessColony(dbTribeClient, dbTerritoryClient, gridConfig, generatorConfig, &islandData, &islandDataLock)
	}
	if generatorConfig.EntityFetchRateInSeconds > 0 {
		go generator.ProcessEntities(dbTribeClient, generatorConfig, &entityData, &entityDataLock, &tribeData, &tribeDataLock)
	}

	http.HandleFunc("/gettribes", getTribes)
	http.HandleFunc("/getdata", getEntities)
	http.HandleFunc("/getislands", getIslands);
	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request){ sendCommand(w, r, dbTribeClient, generatorConfig) } )
	http.HandleFunc("/territoryURL", func(w http.ResponseWriter, r *http.Request){ getTerritoryURL(w, r, generatorConfig) } )
	http.Handle("/", http.FileServer(http.Dir(generatorConfig.StaticDir)))

	endpoint := fmt.Sprintf("%s:%d", generatorConfig.Host, generatorConfig.Port)
	log.Println("Listening on ", endpoint)
	log.Fatal(http.ListenAndServe(endpoint, nil))
}
