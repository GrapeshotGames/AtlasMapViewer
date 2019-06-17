package generator

import (
	"AtlasMapViewer/atlas"
	"image/color"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

var colors = [...]string{
	"lime",
	"blue",
	"yellow",
	"fuchsia",
	"aqua",
	"maroon",
	"red",
	"olive",
	"purple",
	"teal",
}
var colorValues = map[string]color.NRGBA{
	"red":      color.NRGBA{0xff, 0x00, 0x00, 0xff},
	"green":    color.NRGBA{0x00, 0x80, 0x00, 0xff},
	"yellow":   color.NRGBA{0xff, 0xff, 0x00, 0xff},
	"fuchsia":  color.NRGBA{0xff, 0x00, 0x80, 0xff},
	"aqua":     color.NRGBA{0x00, 0xff, 0xff, 0xff},
	"cyan":     color.NRGBA{0x00, 0xff, 0xff, 0xff},
	"blue":     color.NRGBA{0x00, 0x00, 0xff, 0xff},
	"orange":   color.NRGBA{0xff, 0xa5, 0x00, 0xff},
	"purple":   color.NRGBA{0x80, 0x00, 0x80, 0xff},
	"magenta":  color.NRGBA{0xff, 0x00, 0xff, 0xff},
	"lime":     color.NRGBA{0x00, 0xff, 0x00, 0xff},
	"pink":     color.NRGBA{0xff, 0xc0, 0xcb, 0xff},
	"teal":     color.NRGBA{0x00, 0x80, 0x80, 0xff},
	"lavender": color.NRGBA{0xe6, 0xe6, 0xfa, 0xff},
	"brown":    color.NRGBA{0xa5, 0x2a, 0x2a, 0xff},
	"beige":    color.NRGBA{0xf5, 0xf5, 0xdc, 0xff},
	"maroon":   color.NRGBA{0x80, 0x00, 0x00, 0xff},
	"olive":    color.NRGBA{0x80, 0x80, 0x00, 0xff},
	"coral":    color.NRGBA{0xff, 0x7f, 0x50, 0xff},
	"navy":     color.NRGBA{0x00, 0x00, 0x80, 0xff},
	"grey":     color.NRGBA{0xa9, 0xa9, 0xa9, 0xff},
	"black":    color.NRGBA{0x00, 0x00, 0x00, 0xff},
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// ProcessColony runs in a loop processing island info from database
func ProcessColony(tribeDB *redis.Client, territoryDB *redis.Client, gridConfig *atlas.GridConfig, config *Config, islandData *string, islandDataLock *sync.RWMutex) {
	previousCrc := uint32(1)

	for {
		log.Println("Getting island claims")
		counts, crc, err := fetchIslandClaims(territoryDB, gridConfig)
		if err != nil {
			log.Printf("Error! %v\n", err)
		} else if crc == previousCrc {
			log.Println("No CRC changes detected")
		} else {
			previousCrc = crc

			log.Println("Finding top 5 tribes from claims")
			top := TopNTribes(5, counts)
			for i := 0; i < len(top); i++ {
				tribe := (*counts)[top[i]]
				color := colorValues[colors[i]]
				for _, island := range tribe.islands {
					island.Color = color
					island.ColorName = colors[i]
				}
			}

			log.Println("Generating island data")
			virtualPixels := int(gridConfig.GridSize) * Max(gridConfig.TotalGridsX, gridConfig.TotalGridsY)
			generateIslandData(counts, virtualPixels, islandData, islandDataLock)
		}

		log.Println("Done, waiting till next round")
		time.Sleep(time.Duration(config.ColonyFetchRateInSeconds) * time.Second)
	}
}
