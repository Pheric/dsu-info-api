package main

import (
	"fmt"

	"github.com/the-rileyj/dsu-info-api/sodexo"
)

// @title DSU Info API
// @version 0.0.1
// @description DSU information API

func main() {
	// router := gin.Default()

	// router.Run(":1776")
	fmt.Println(sodexo.GetTodaysMenu())
}
