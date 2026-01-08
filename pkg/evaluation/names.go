package evaluation

import (
	"fmt"
	"math/rand/v2"
)

var (
	adjectives = []string{
		"swift", "bright", "calm", "bold", "keen",
		"fair", "warm", "cool", "sharp", "clear",
		"quick", "smart", "fresh", "light", "soft",
		"vivid", "eager", "agile", "clever", "gentle",
	}
	nouns = []string{
		"falcon", "river", "cloud", "spark", "wave",
		"bloom", "stone", "flame", "brook", "maple",
		"cedar", "frost", "dawn", "dusk", "peak",
		"grove", "reef", "mesa", "vale", "cove",
	}
)

// GenerateRunName creates a memorable name for an evaluation run.
func GenerateRunName() string {
	adj := adjectives[rand.IntN(len(adjectives))]
	noun := nouns[rand.IntN(len(nouns))]
	num := rand.IntN(1000)

	return fmt.Sprintf("%s-%s-%03d", adj, noun, num)
}
