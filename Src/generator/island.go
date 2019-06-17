package generator

import (
	"encoding/json"
	"encoding/binary"
	"image/color"
	"time"
	"strings"
	"hash/crc32"
	"log"
	"sync"

	"AtlasMapViewer/atlas"

	"github.com/go-redis/redis"
)

// IslandClaim represents json stored in Redis (and calculated internals) for each island
type IslandClaim struct {
	IslandID                       int         `json:"islandId"`
	SettlementFlagName             string      `json:"settlementFlagName"`
	OwnerTribeID                   uint64      `json:"ownerTribeId"`
	OwnerName                      string      `json:"ownerName"`
	CombatPhaseStartTime           int         `json:"combatPhaseStartTime"`
	LastUTCTimeAdjustedCombatPhase int         `json:"lastUTCTimeAdjustedCombatPhase"`
	TaxRate                        float64     `json:"taxRate"`
	BIsContested                   bool        `json:"bIsContested"`
	NumSettlers                    int         `json:"numSettlers"`
	X                              float64     `json:"-"`
	Y                              float64     `json:"-"`
	Radius                         float64     `json:"-"`
	Color                          color.NRGBA `json:"-"`
	ColorName                      string      `json:"-"`
	IslandPoints                   int         `json:"-"`
	WarringTribeID                 uint64      `json:"-"`
	WarStartUTC                    uint32      `json:"-"`
	WarEndUTC                      uint32      `json:"-"`
}

// WarDeclaration represents json stored in Redis for war declarations
type WarDeclaration struct {
	IslandID       int    `json:"islandId"`
	WarringTribeID uint64 `json:"warringTribeID"`
	WarStartUTC    uint32 `json:"warStartUTC"`
	WarEndUTC      uint32 `json:"warEndUTC"`
}

// IslandInfoOutput json for front-end consumption
type IslandInfoOutput struct {
	IslandID             int     `json:"IslandID"`
	X                    float64 `json:"X"`
	Y                    float64 `json:"Y"`
	Size                 float64 `json:"Size"`
	TribeID              uint64  `json:"TribeId"`
	Color                string  `json:"Color"`
	IslandPoints         int     `json:"IslandPoints"`
	SettlementName       string  `json:"SettlementName"`
	TaxRate              float64 `json:"TaxRate"`
	CombatPhaseStartTime int     `json:"CombatPhaseStartTime"`
	WarringTribeID       uint64  `json:"WarringTribeID"`
	WarStartUTC          uint32  `json:"WarStartUTC"`
	WarEndUTC            uint32  `json:"WarEndUTC"`
	NumSettlers          int     `json:"NumSettlers"`
}

// CompanyInfoOutput json for front-end consumption
type CompanyInfoOutput struct {
	TribeID   uint64  `json:"TribeId"`
	TribeName string  `json:"TribeName"`
	FlagURL   *string `json:"FlagURL"`
}

// IslandOutput json for front-end consumption
type IslandOutput struct {
	Version   int64               `json:"version"`
	Islands   []IslandInfoOutput  `json:"Islands"`
	Companies []CompanyInfoOutput `json:"Companies"`
}

func generateIslandData(tribes *map[uint64]*TribeCount, virtualPixels int, islandData *string, islandDataLock *sync.RWMutex) {
	output := IslandOutput{
		Version:   time.Now().Unix(),
		Islands:   make([]IslandInfoOutput, 0),
		Companies: make([]CompanyInfoOutput, 0),
	}
	for _, tribe := range *tribes {
		for _, island := range tribe.islands {
			islandOut := IslandInfoOutput{
				IslandID:             island.IslandID,
				X:                    island.X / float64(virtualPixels),
				Y:                    island.Y / float64(virtualPixels),
				TribeID:              island.OwnerTribeID,
				Size:                 island.Radius / float64(virtualPixels),
				Color:                island.ColorName,
				IslandPoints:         island.IslandPoints,
				SettlementName:       island.SettlementFlagName,
				TaxRate:              island.TaxRate,
				CombatPhaseStartTime: island.CombatPhaseStartTime,
				WarringTribeID:       island.WarringTribeID,
				WarStartUTC:          island.WarStartUTC,
				WarEndUTC:            island.WarEndUTC,
				NumSettlers:          island.NumSettlers,
			}
			output.Islands = append(output.Islands, islandOut)
			if len(tribe.name) == 0 {
				tribe.name = island.OwnerName
			}
		}
		companyOut := CompanyInfoOutput{
			TribeID:   tribe.tribeID,
			TribeName: tribe.name,
			FlagURL:   tribe.flagURL,
		}
		output.Companies = append(output.Companies, companyOut)
	}

	// save off the json
	js, _ := json.Marshal(output)
	islandDataLock.Lock()
	*islandData = string(js)
	islandDataLock.Unlock()
}

func fixBadString(s string) string {
	var out string
	for _, c := range s {
		if c >= 32 && c <= 126 {
			out += string(c)
		} else {
			out += "[]"
		}
	}
	return out
}

func removeAndFixSettlementName(in string) (string, *string) {
	idxStart := strings.Index(in, ",\"settlementFlagName\"")
	if idxStart == -1 {
		return in, nil
	}
	idxEnd := strings.Index(in, ",\"ownerTribeId\"")
	if idxEnd == -1 {
		return in, nil
	}

	fixedName := fixBadString(in[idxStart+23 : idxEnd-1])
	fixedJSON := in[:idxStart] + in[idxEnd:]
	return fixedJSON, &fixedName
}

func removeAndFixOwnerName(in string) (string, *string) {
	idxStart := strings.Index(in, ",\"ownerName\"")
	if idxStart == -1 {
		return in, nil
	}
	idxEnd := strings.Index(in, ",\"combatPhaseStartTime\"")
	if idxEnd == -1 {
		return in, nil
	}

	fixedName := fixBadString(in[idxStart+14 : idxEnd-1])
	fixedJSON := in[:idxStart] + in[idxEnd:]
	return fixedJSON, &fixedName
}

func fetchIslandClaims(db *redis.Client, grid *atlas.GridConfig) (*map[uint64]*TribeCount, uint32, error) {
	hash := crc32.NewIEEE()
	islands := make(map[int]*IslandClaim)
	countPerTribe := make(map[uint64]*TribeCount)
	grey := colorValues["grey"]

	raw, err := db.HGetAll("islands").Result()
	if err != nil {
		return nil, 0, err
	}
	for _, v := range raw {
		// add to CRC
		binary.Write(hash, binary.LittleEndian, []byte(v))

		// decode island claim json
		islandClaim := IslandClaim{
			NumSettlers: -1,
		}
		err := json.Unmarshal([]byte(v), &islandClaim)
		if err != nil {
			// UE4 is writing bad strings into JSON :(
			tmp, fixedSettlementName := removeAndFixSettlementName(v)
			tmp, fixedOwnerName := removeAndFixOwnerName(tmp)
			err := json.Unmarshal([]byte(tmp), &islandClaim)
			if err != nil {
				log.Printf("Error Parsing Island Claim! %v\n", err)
				continue
			}
			if fixedSettlementName != nil {
				islandClaim.SettlementFlagName = *fixedSettlementName
			} else {
				islandClaim.SettlementFlagName = "<unknown>"
			}
			if fixedOwnerName != nil {
				islandClaim.OwnerName = *fixedOwnerName
			} else {
				islandClaim.OwnerName = "<unknown>"
			}
		}
		if islandClaim.OwnerTribeID == 0 {
			continue
		}
		island := grid.Islands[islandClaim.IslandID]
		if island == nil || island.IslandPoints <= 0 {
			continue
		}

		// add island position and default color
		islandClaim.X = island.WorldX
		islandClaim.Y = island.WorldY
		islandClaim.Radius = (island.IslandWidth + island.IslandHeight) / 4
		islandClaim.Color = grey
		islandClaim.ColorName = "grey"
		islandClaim.IslandPoints = island.IslandPoints

		// temp map to connect war declartions
		islands[islandClaim.IslandID] = &islandClaim

		// sum per tribe island points
		tribe := countPerTribe[islandClaim.OwnerTribeID]
		if tribe != nil {
			tribe.count += island.IslandPoints
			tribe.islands = append(tribe.islands, &islandClaim)
		} else {
			tribe = &TribeCount{
				tribeID: islandClaim.OwnerTribeID,
				count:   island.IslandPoints,
				islands: []*IslandClaim{&islandClaim},
			}
			countPerTribe[islandClaim.OwnerTribeID] = tribe
		}
	}

	raw, err = db.HGetAll("islands.war").Result()
	if err != nil {
		goto End
	}
	for _, v := range raw {
		// add to CRC
		binary.Write(hash, binary.LittleEndian, []byte(v))

		// decode war declaration
		warDeclaration := WarDeclaration{}
		err := json.Unmarshal([]byte(v), &warDeclaration)
		if err != nil {
			log.Println("Invalid Json: " + v)
			continue
		}
		if warDeclaration.WarringTribeID == 0 || warDeclaration.WarStartUTC == 0 || warDeclaration.WarEndUTC == 0 {
			continue
		}

		// copy values onto the island
		if island, found := islands[warDeclaration.IslandID]; found {
			island.WarringTribeID = warDeclaration.WarringTribeID
			island.WarStartUTC = warDeclaration.WarStartUTC
			island.WarEndUTC = warDeclaration.WarEndUTC
		}
	}

End:
	return &countPerTribe, hash.Sum32(), nil
}
