package types

const (
	EngineRedis  = "redis"
	EngineValkey = "valkey"
)

var SupportedCacheEngines = []string{EngineRedis, EngineValkey}

func IsSupportedCacheEngine(engine string) bool {
	if engine == "" {
		return true
	}
	for _, supported := range SupportedCacheEngines {
		if engine == supported {
			return true
		}
	}
	return false
}

func EngineDisplayName(engine string) string {
	if engine == EngineValkey {
		return "Valkey"
	}
	return "Redis"
}
