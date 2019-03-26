package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/the-rileyj/dsu-info-api/trojantimes"
)

// @title DSU Info API
// @version 0.0.1
// @description DSU information API

func main() {
	db, err := gorm.Open("postgres", "host=localhost port=8899 user=postgres password=mysecretpassword sslmode=disable")

	if err != nil {
		panic(err)
	}

	defer db.Close()

	db.DropTableIfExists(
		&trojantimes.TrojanTimesArticle{},
		&trojantimes.ArticleImages{},
		&trojantimes.ArticleBody{},
		&trojantimes.ArticleComment{},
		&trojantimes.Category{},
		&trojantimes.Tag{},
	)

	db.CreateTable(
		&trojantimes.TrojanTimesArticle{},
		&trojantimes.ArticleImages{},
		&trojantimes.ArticleBody{},
		&trojantimes.ArticleComment{},
		&trojantimes.Category{},
		&trojantimes.Tag{},
	)

	go trojantimes.ScrapeTrojanTimesArticlesWithDatabase(db)

	router := gin.Default()

	router.NoRoute(func(c *gin.Context) {
		c.AbortWithStatusJSON(
			404,
			gin.H{
				"err": fmt.Sprintf(`requested path "%s" does not exist`, c.Request.URL.Path),
			},
		)
	})

	router.GET("/article", func(c *gin.Context) {
		var (
			searchTrojanTimesArticle trojantimes.TrojanTimesArticle
			trojanTimesArticle       trojantimes.TrojanTimesArticle
		)

		searchTrojanTimesArticle.Author = c.Query("author")

		id, err := strconv.Atoi(c.Query("id"))

		if err != nil {
			searchTrojanTimesArticle.ID = 0
		} else {
			searchTrojanTimesArticle.ID = id
		}

		err = db.Preload("Body").Preload("Comments").Preload("Categories").Preload("Images").Preload("Tags").First(&trojanTimesArticle, searchTrojanTimesArticle).Error

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": err.Error(),
				},
			)

			return
		}

		c.JSON(
			200,
			trojanTimesArticle,
		)
	})

	router.GET("/articles", func(c *gin.Context) {
		var (
			trojanTimesArticles      []trojantimes.TrojanTimesArticle
			searchTrojanTimesArticle trojantimes.TrojanTimesArticle
			skimTrojanTimesArticles  []trojantimes.SkimTrojanTimesArticle = make([]trojantimes.SkimTrojanTimesArticle, 0)
		)

		searchTrojanTimesArticle.Author = c.Query("author")

		err = db.Find(&trojanTimesArticles, searchTrojanTimesArticle).Error

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": err.Error(),
				},
			)

			return
		}

		for _, trojanTimesArticle := range trojanTimesArticles {
			skimTrojanTimesArticles = append(skimTrojanTimesArticles, trojanTimesArticle.ToSkim())
		}

		c.JSON(
			200,
			skimTrojanTimesArticles,
		)
	})

	router.GET("/categories", func(c *gin.Context) {
		var categories []trojantimes.Category

		// GORM translates CamelCase variable names into snake_case,
		// so we use category_name here instead of CategoryName
		err = db.Select("DISTINCT(category_name)").Find(&categories).Error

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": err.Error(),
				},
			)

			return
		}

		c.JSON(
			200,
			categories,
		)
	})

	router.GET("/articles/category", func(c *gin.Context) {
		var (
			articles       = make([]*trojantimes.TrojanTimesArticle, 0)
			searchCategory trojantimes.Category
			categories     []trojantimes.Category
		)

		id, err := strconv.Atoi(c.DefaultQuery("categoryid", "0"))

		if id < 0 {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": "id must be a positive integer",
				},
			)

			return
		}

		searchCategory.ID = uint(id)
		searchCategory.CategoryName = c.DefaultQuery("categoryname", "")

		// Prepopulate the article results
		err = db.Where(searchCategory).
			Preload("Articles").
			Preload("Articles.Body").
			Preload("Articles.Comments").
			Preload("Articles.Categories").
			Preload("Articles.Images").
			Preload("Articles.Tags").Find(&categories).Error

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": err.Error(),
				},
			)

			return
		}

		for _, category := range categories {
			articles = append(articles, category.Articles...)
		}

		c.JSON(
			200,
			articles,
		)
	})

	// Search article via list of categories
	type searchCategories struct {
		categories []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		}
	}

	router.GET("/articles/category", func(c *gin.Context) {
		var (
			articles       = make([]*trojantimes.TrojanTimesArticle, 0)
			searchCategory trojantimes.Category
			categories     []trojantimes.Category
		)

		id, err := strconv.Atoi(c.DefaultQuery("categoryid", "0"))

		if id < 0 {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": "id must be a positive integer",
				},
			)

			return
		}

		searchCategory.ID = uint(id)
		searchCategory.CategoryName = c.DefaultQuery("categoryname", "")

		// Prepopulate the article results
		err = db.Where(searchCategory).
			Preload("Articles").
			Preload("Articles.Body").
			Preload("Articles.Comments").
			Preload("Articles.Categories").
			Preload("Articles.Images").
			Preload("Articles.Tags").Find(&categories).Error

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": err.Error(),
				},
			)

			return
		}

		for _, category := range categories {
			articles = append(articles, category.Articles...)
		}

		c.JSON(
			200,
			articles,
		)
	})

	router.GET("/tags", func(c *gin.Context) {
		var tags []trojantimes.Tag

		// GORM translates CamelCase variable names into snake_case,
		// so we use tag_name here instead of TagName
		err = db.Select("DISTINCT(tag_name)").Find(&tags).Error

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": err.Error(),
				},
			)

			return
		}

		c.JSON(
			200,
			tags,
		)
	})

	router.GET("/articles/tag", func(c *gin.Context) {
		var (
			articles  = make([]*trojantimes.TrojanTimesArticle, 0)
			searchTag trojantimes.Tag
			tags      []trojantimes.Tag
		)

		id, err := strconv.Atoi(c.DefaultQuery("tagid", "0"))

		if id < 0 {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": "id must be a positive integer",
				},
			)

			return
		}

		searchTag.ID = uint(id)
		searchTag.TagName = c.Query("tagname")

		// Prepopulate the article results
		err = db.Where(searchTag).
			Preload("Articles").
			Preload("Articles.Body").
			Preload("Articles.Comments").
			Preload("Articles.Categories").
			Preload("Articles.Images").
			Preload("Articles.Tags").Find(&tags).Error

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"err": err.Error(),
				},
			)

			return
		}

		for _, tags := range tags {
			articles = append(articles, tags.Articles...)
		}

		c.JSON(
			200,
			articles,
		)
	})

	router.Run(":1776")
}
