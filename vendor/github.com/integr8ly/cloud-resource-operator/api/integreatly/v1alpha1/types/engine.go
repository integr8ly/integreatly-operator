package types

const (
	EngineRedis  = "redis"
	EngineValkey = "valkey"
)

var SupportedRedisEngines = []string{EngineRedis, EngineValkey}

func IsSupportedRedisEngine(engine string) bool {
	if engine == "" {
		return true
	}
	for _, supported := range SupportedRedisEngines {
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
