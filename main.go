package main

import (
	"fmt"

	"github.com/the-rileyj/dsu-info-api/trojantimes"
)

// @title DSU Info API
// @version 0.0.1
// @description DSU information API

func main() {
	// router := gin.Default()

	// router.Run(":1776")
	c := trojantimes.GetTrojanTimesArticlesWithInfo(map[string]trojantimes.TrojanTimesArticle{})

	fmt.Println(<-c)
}
