package main

import (
	_ "github.com/Zhiruosama/ai_nexus/configs"
	app "github.com/Zhiruosama/ai_nexus/internal"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/db"
)

func main() {
	app.Run()
}
