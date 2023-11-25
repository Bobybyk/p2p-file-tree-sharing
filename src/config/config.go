package config

var Debug = false
var DebugSpam = false

func SetDebug(val bool) {
	Debug = val
}

func SetDebugSpam(val bool) {
	DebugSpam = val
}
