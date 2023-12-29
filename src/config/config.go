package config

var Debug = true
var DebugSpam = false
var ClientName = "ogu"

func SetDebug(val bool) {
	Debug = val
}

func SetDebugSpam(val bool) {
	DebugSpam = val
}
