package pick

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Word lists for random subdomain generation
var adjectives = []string{
	"brave", "calm", "dark", "eager", "fair", "glad", "happy", "idle", "keen", "lush",
	"mild", "neat", "open", "pure", "quick", "rich", "safe", "tall", "vast", "warm",
	"able", "bold", "cool", "deep", "even", "fast", "gold", "high", "just", "kind",
	"lean", "main", "nice", "pale", "rare", "slim", "true", "used", "wide", "wise",
}

var nouns = []string{
	"apex", "beam", "cave", "dawn", "edge", "fern", "gate", "haze", "iris", "jade",
	"kite", "lake", "mesa", "node", "onyx", "pine", "quay", "reef", "star", "tide",
	"vale", "wave", "yard", "zinc", "arch", "bark", "cove", "dune", "flux", "glen",
	"hive", "isle", "jazz", "knot", "loom", "moss", "nest", "opal", "peak", "rift",
}

func pickRandom(words []string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
	return words[n.Int64()]
}

// RandomSubdomain generates a random three-word subdomain like "brave-apex-dawn".
func RandomSubdomain() string {
	return fmt.Sprintf("%s-%s-%s", pickRandom(adjectives), pickRandom(nouns), pickRandom(nouns))
}
