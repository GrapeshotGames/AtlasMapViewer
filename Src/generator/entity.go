package generator

import (
	"encoding/binary"
	"encoding/json"
	"log"
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

// ServerLocation relative percentage to a specific server's origin.
type ServerLocation struct {
	ServerID                [2]uint16
	ServerXRelativeLocation float64
	ServerYRelativeLocation float64
}

func ProcessEntities(client *redis.Client, config *Config, entityData *string, entityDataLock *sync.RWMutex, tribeData *string, tribeDataLock *sync.RWMutex) {
	var kidsWithBadParents map[string]bool
	kidsWithBadParents = make(map[string]bool)
	_ = kidsWithBadParents

	for {
		tribes := make(map[string]string)
		entities := make(map[string]EntityInfo)
		_ = entities

		for _, record := range scan(client, "tribedata:*") {
			tribes[record["TribeID"]] = record["TribeName"]
		}

		js, _ := json.Marshal(tribes)
		tribeDataLock.Lock()
		*tribeData = string(js)
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

		js, _ = json.Marshal(entities)
		entityDataLock.Lock()
		*entityData = string(js)
		entityDataLock.Unlock()

		time.Sleep(time.Duration(config.EntityFetchRateInSeconds) * time.Second)
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

	// Batch fetch each entity to avoid overwhelming redis
	results := make(map[string]*redis.StringStringMapCmd)
	batch := 0
	pipe := client.Pipeline()
	for i := 0; i < len(keys); i++ {
		results[keys[i]] = pipe.HGetAll(keys[i])
		batch++
		if batch > 2000 {
			pipe.Exec()		
			batch = 0
			pipe = client.Pipeline()
		}
	}
	if batch > 0 {
		pipe.Exec()
	}
	for _, id := range keys {
		var err error
		records[id], err = results[id].Result()
		if err != nil {
			log.Fatal(err)
		}
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