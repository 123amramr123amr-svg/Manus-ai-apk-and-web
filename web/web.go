// Package web bundles the frontends that the Go server serves: the Manus API
// console (static/) and the standalone on-device agent (agent/).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var assets embed.FS

//go:embed all:agent
var agentAssets embed.FS

// StaticFS returns the console frontend rooted at the static directory.
func StaticFS() fs.FS {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return sub
}

// AgentFS returns the standalone agent frontend rooted at the agent directory.
// The very same files are copied into the Android APK assets by
// tools/syncassets, so the phone app works without any server.
func AgentFS() fs.FS {
	sub, err := fs.Sub(agentAssets, "agent")
	if err != nil {
		panic(err)
	}
	return sub
}
