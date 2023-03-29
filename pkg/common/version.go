package common

// NAME of the App
var NAME = "satokens"

// SUMMARY of the Version
var SUMMARY = "0.0.0"

// BRANCH of the Version
var BRANCH = "main"

// VERSION of Release
var VERSION = "0.0.0"

var COMMIT = "dirty"

// AppVersion --
var AppVersion AppVersionInfo

// AppVersionInfo --
type AppVersionInfo struct {
	Name    string
	Version string
	Branch  string
	Summary string
	Commit  string
}

func init() {
	AppVersion = AppVersionInfo{
		Name:    NAME,
		Version: VERSION,
		Branch:  BRANCH,
		Summary: SUMMARY,
		Commit:  COMMIT,
	}
}
