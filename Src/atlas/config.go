package atlas

import (
	"encoding/json"
	"math"
	"os"
	"time"
)

// RedisConfig holds the redis connect configuration
type RedisConfig struct {
	Name     string `json:"Name"`
	URL      string `json:"URL"`
	Port     int    `json:"Port"`
	Password string `json:"Password"`
}

// SeverOnlyConfig holds the server-only, i.e. hidden from client, Atlas cluster configuration.  Mostly DB and AWS configuration.
type SeverOnlyConfig struct {
	LocalS3URL         string `json:"LocalS3URL"`
	LocalS3AccessKeyID string `json:"LocalS3AccessKeyId"`
	LocalS3SecretKey   string `json:"LocalS3SecretKey"`
	LocalS3BucketName  string `json:"LocalS3BucketName"`
	LocalS3Region      string `json:"LocalS3Region"`
	TribeLogConfig     struct {
		MaxRedisEntries int    `json:"MaxRedisEntries"`
		BackupMode      string `json:"BackupMode"`
		MaxFileHistory  int    `json:"MaxFileHistory"`
		HTTPBackupURL   string `json:"HttpBackupURL"`
		HTTPAPIKey      string `json:"HttpAPIKey"`
		S3KeyPrefix     string `json:"S3KeyPrefix"`
	} `json:"TribeLogConfig"`
	SharedLogConfig struct {
		FetchRateSec            int    `json:"FetchRateSec"`
		SnapshotCleanupSec      int    `json:"SnapshotCleanupSec"`
		SnapshotRateSec         int    `json:"SnapshotRateSec"`
		SnapshotExpirationHours int    `json:"SnapshotExpirationHours"`
		BackupMode              string `json:"BackupMode"`
		MaxFileHistory          int    `json:"MaxFileHistory"`
		HTTPBackupURL           string `json:"HttpBackupURL"`
		HTTPAPIKey              string `json:"HttpAPIKey"`
		S3KeyPrefix             string `json:"S3KeyPrefix"`
	} `json:"SharedLogConfig"`
	TravelDataConfig struct {
		BackupMode     string `json:"BackupMode"`
		MaxFileHistory int    `json:"MaxFileHistory"`
		HTTPBackupURL  string `json:"HttpBackupURL"`
		HTTPAPIKey     string `json:"HttpAPIKey"`
		S3URL          string `json:"S3URL"`
		S3AccessKeyID  string `json:"S3AccessKeyId"`
		S3SecretKey    string `json:"S3SecretKey"`
		S3BucketName   string `json:"S3BucketName"`
		S3KeyPrefix    string `json:"S3KeyPrefix"`
	} `json:"TravelDataConfig"`
	DatabaseConnections []RedisConfig `json:"DatabaseConnections"`
}

// LoadSeverOnlyConfig loads and returns a SeverOnlyConfig from the specified file
func LoadSeverOnlyConfig(path string) (result *SeverOnlyConfig, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg SeverOnlyConfig
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetDatabaseByName looks up a database config by name
func (c *SeverOnlyConfig) GetDatabaseByName(name string) (RedisConfig, bool) {
	for _, v := range c.DatabaseConnections {
		if v.Name == name {
			return v, true
		}
	}
	return RedisConfig{
		Name:     "<not found>",
		URL:      "localhost",
		Port:     6379,
		Password: "foobared",
	}, false
}

type IslandInstance struct {
	Name                                     string  `json:"name"`
	ID                                       int     `json:"id"`
	MinTreasureQuality                       float64 `json:"minTreasureQuality"`
	MaxTreasureQuality                       float64 `json:"maxTreasureQuality"`
	UseNpcVolumesForTreasures                bool    `json:"useNpcVolumesForTreasures"`
	UseLevelBoundsForTreasures               bool    `json:"useLevelBoundsForTreasures"`
	PrioritizeVolumesForTreasures            bool    `json:"prioritizeVolumesForTreasures"`
	IslandPoints                             int     `json:"islandPoints"`
	IslandTreasureBottleSupplyCrateOverrides string  `json:"islandTreasureBottleSupplyCrateOverrides"`
	IslandWidth                              float64 `json:"islandWidth"`
	IslandHeight                             float64 `json:"islandHeight"`
	WorldX                                   float64 `json:"worldX"`
	WorldY                                   float64 `json:"worldY"`
	Rotation                                 float64 `json:"rotation"`
}

// ServerGridConfig holds the per-server game information for an Atlas cluster
type ServerGridConfig struct {
	GridX                                                  int               `json:"gridX"`
	GridY                                                  int               `json:"gridY"`
	MachineIDTag                                           string            `json:"MachineIdTag"`
	IP                                                     string            `json:"ip"`
	Name                                                   string            `json:"name"`
	Port                                                   int               `json:"port"`
	GamePort                                               int               `json:"gamePort"`
	SeamlessDataPort                                       int               `json:"seamlessDataPort"`
	IsHomeServer                                           bool              `json:"isHomeServer"`
	AdditionalCmdLineParams                                string            `json:"AdditionalCmdLineParams"`
	OverrideShooterGameModeDefaultGameIni                  map[string]string `json:"OverrideShooterGameModeDefaultGameIni"`
	FloorZDist                                             int               `json:"floorZDist"`
	UtcOffset                                              int               `json:"utcOffset"`
	TransitionMinZ                                         int               `json:"transitionMinZ"`
	GlobalBiomeSeamlessServerGridPreOffsetValues           string            `json:"GlobalBiomeSeamlessServerGridPreOffsetValues"`
	GlobalBiomeSeamlessServerGridPreOffsetValuesOceanWater string            `json:"GlobalBiomeSeamlessServerGridPreOffsetValuesOceanWater"`
	OceanDinoDepthEntriesOverride                          string            `json:"OceanDinoDepthEntriesOverride"`
	OceanEpicSpawnEntriesOverrideValues                    string            `json:"OceanEpicSpawnEntriesOverrideValues"`
	OceanFloatsamCratesOverride                            string            `json:"oceanFloatsamCratesOverride"`
	TreasureMapLootTablesOverride                          string            `json:"treasureMapLootTablesOverride"`
	OceanEpicSpawnEntriesOverrideTemplateName              string            `json:"oceanEpicSpawnEntriesOverrideTemplateName"`
	NPCShipSpawnEntriesOverrideTemplateName                string            `json:"NPCShipSpawnEntriesOverrideTemplateName"`
	RegionOverrides                                        string            `json:"regionOverrides"`
	WaterColorR                                            float64           `json:"waterColorR"`
	WaterColorG                                            float64           `json:"waterColorG"`
	WaterColorB                                            float64           `json:"waterColorB"`
	SkyStyleIndex                                          int               `json:"skyStyleIndex"`
	ServerIslandPointsMultiplier                           float64           `json:"serverIslandPointsMultiplier"`
	ServerCustomDatas1                                     string            `json:"ServerCustomDatas1,omitempty"`
	ServerCustomDatas2                                     string            `json:"ServerCustomDatas2,omitempty"`
	ClientCustomDatas1                                     string            `json:"ClientCustomDatas1"`
	ClientCustomDatas2                                     string            `json:"ClientCustomDatas2"`
	Sublevels                                              []struct {
		Name                      string  `json:"name"`
		AdditionalTranslationX    float64 `json:"additionalTranslationX"`
		AdditionalTranslationY    float64 `json:"additionalTranslationY"`
		AdditionalTranslationZ    float64 `json:"additionalTranslationZ"`
		AdditionalRotationPitch   float64 `json:"additionalRotationPitch"`
		AdditionalRotationYaw     float64 `json:"additionalRotationYaw"`
		AdditionalRotationRoll    float64 `json:"additionalRotationRoll"`
		ID                        int     `json:"id"`
		LandscapeMaterialOverride int     `json:"landscapeMaterialOverride"`
	} `json:"sublevels"`
	LastModified        time.Time        `json:"lastModified"`
	LastImageOverride   string           `json:"lastImageOverride"`
	IslandLocked        bool             `json:"islandLocked"`
	DiscoLocked         bool             `json:"discoLocked"`
	PathsLocked         bool             `json:"pathsLocked"`
	ExtraSublevels      []string         `json:"extraSublevels"`
	TotalExtraSublevels []string         `json:"totalExtraSublevels"`
	IslandInstances     []IslandInstance `json:"islandInstances"`
	DiscoZones          []struct {
		Name              string  `json:"name"`
		SizeX             float64 `json:"sizeX"`
		SizeY             float64 `json:"sizeY"`
		SizeZ             float64 `json:"sizeZ"`
		ID                int     `json:"id"`
		Xp                float64 `json:"xp"`
		BIsManuallyPlaced bool    `json:"bIsManuallyPlaced"`
		ManualVolumeName  string  `json:"ManualVolumeName,omitempty"`
		ExplorerNoteIndex int     `json:"explorerNoteIndex"`
		AllowSea          bool    `json:"allowSea"`
		WorldX            float64 `json:"worldX"`
		WorldY            float64 `json:"worldY"`
		Rotation          float64 `json:"rotation"`
	} `json:"discoZones"`
	SpawnRegions       []interface{} `json:"spawnRegions"`
	ServerTemplateName string        `json:"serverTemplateName"`
}

// GridConfig holds the game configuration for an Atlas cluster
type GridConfig struct {
	BaseServerArgs                        string             `json:"BaseServerArgs"`
	GridSize                              float64            `json:"gridSize"`
	MetaWorldURL                          string             `json:"MetaWorldURL"`
	WorldFriendlyName                     string             `json:"WorldFriendlyName"`
	WorldAtlasID                          string             `json:"WorldAtlasId"`
	AuthListURL                           string             `json:"AuthListURL"`
	WorldAtlasPassword                    string             `json:"WorldAtlasPassword"`
	ModIDs                                string             `json:"ModIDs"`
	MapImageURL                           string             `json:"MapImageURL"`
	TotalGridsX                           int                `json:"totalGridsX"`
	TotalGridsY                           int                `json:"totalGridsY"`
	BUseUTCTime                           bool               `json:"bUseUTCTime"`
	ColumnUTCOffset                       float64            `json:"columnUTCOffset"`
	Day0                                  string             `json:"Day0"`
	GlobalTransitionMinZ                  float64            `json:"globalTransitionMinZ"`
	AdditionalCmdLineParams               string             `json:"AdditionalCmdLineParams"`
	OverrideShooterGameModeDefaultGameIni map[string]string  `json:"OverrideShooterGameModeDefaultGameIni"`
	GlobalGameplaySetup                   string             `json:"globalGameplaySetup"`
	CoordsScaling                         float64            `json:"coordsScaling"`
	ShowServerInfo                        bool               `json:"showServerInfo"`
	ShowDiscoZoneInfo                     bool               `json:"showDiscoZoneInfo"`
	ShowShipPathsInfo                     bool               `json:"showShipPathsInfo"`
	ShowIslandNames                       bool               `json:"showIslandNames"`
	ShowBackground                        bool               `json:"showBackground"`
	BackgroundImgPath                     string             `json:"backgroundImgPath"`
	DiscoZonesImagePath                   string             `json:"discoZonesImagePath"`
	Servers                               []ServerGridConfig `json:"servers"`
	SpawnerOverrideTemplates              []struct {
		Name                           string  `json:"Name"`
		NPCSpawnEntries                string  `json:"NPCSpawnEntries"`
		NPCSpawnLimits                 string  `json:"NPCSpawnLimits"`
		MaxDesiredNumEnemiesMultiplier float64 `json:"MaxDesiredNumEnemiesMultiplier"`
	} `json:"spawnerOverrideTemplates"`
	IDGenerator          int `json:"idGenerator"`
	RegionsIDGenerator   int `json:"regionsIdGenerator"`
	ShipPathsIDGenerator int `json:"shipPathsIdGenerator"`
	ShipPaths            []struct {
		Nodes []struct {
			ControlPointsDistance float64 `json:"controlPointsDistance"`
			WorldX                float64 `json:"worldX"`
			WorldY                float64 `json:"worldY"`
			Rotation              float64 `json:"rotation"`
		} `json:"Nodes"`
		PathID                    int     `json:"PathId"`
		IsLooping                 bool    `json:"isLooping"`
		PathName                  string  `json:"PathName"`
		AutoSpawnShipClass        string  `json:"AutoSpawnShipClass"`
		AutoSpawnEveryUTCInterval float64 `json:"AutoSpawnEveryUTCInterval"`
		AutoSpawn                 bool    `json:"autoSpawn"`
	} `json:"shipPaths"`
	LastImageOverride string                  `json:"lastImageOverride"`
	ServerTemplates   []interface{}           `json:"serverTemplates"`
	Islands           map[int]*IslandInstance `json:"-"`
}

// LoadGridConfig loads and returns a GridConfig from the specified file
func LoadGridConfig(path string) (result *GridConfig, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg GridConfig
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	// fix up island points
	cfg.Islands = make(map[int]*IslandInstance)
	for i := 0; i < len(cfg.Servers); i++ {
		server := &cfg.Servers[i]
		for j := 0; j < len(server.IslandInstances); j++ {
			island := &server.IslandInstances[j]
			cfg.Islands[island.ID] = island
			if island.IslandPoints <= 0 {
				island.IslandPoints = -1
				continue
			} else if island.IslandPoints == 1 {
				tmp := int(math.Pow(island.IslandWidth*island.IslandHeight, 0.6) * 0.000015)
				if tmp < 1 {
					tmp = 1
				} else if tmp > 100 {
					tmp = 100
				}
				island.IslandPoints = int(math.Ceil(server.ServerIslandPointsMultiplier * float64(tmp)))
			}
		}
	}

	return &cfg, nil
}
