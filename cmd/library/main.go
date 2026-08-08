package main

import (
	"RSOI_PROJECT/internal/librarydb"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type libraryConfig struct {
	queries *librarydb.Queries
	client  http.Client
}

func main() {

	if err := godotenv.Load(); err != nil {
		fmt.Println("error loading .env")
	}

	db_url := os.Getenv("LIBRARIES_DB_URL")
	db, err := sql.Open("postgres", db_url)
	if err != nil {
		fmt.Println("error openning db")
	}
	dbQueries := librarydb.New(db)

	myClient := http.Client{
		Timeout: 10 * time.Second,
	}

	cfg := libraryConfig{
		queries: dbQueries,
		client:  myClient,
	}

	server := gin.Default()
	server.GET("/api/v1/libraries", cfg.getLibraries)
	// server.GET("/api/v1/libraries/:libraryUid", getLibrary)
	// server.GET("/api/v1/libraries/:libraryUid/books", getLibraryBooks)
	// server.GET("/api/v1/libraries/:libraryUid/books/:bookUid", getLibraryBook)
	// server.POST("/api/v1/libraries/:libraryUid/books/:bookUid/decrease", decreaseBookCount)
	// server.POST("/api/v1/libraries/:libraryUid/books/:bookUid/increase", increaseBookCount)
	// server.GET("/manage/health", healthCheck)

	log.Println("Library service starting on port:8060")
	if err := server.Run(":8060"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (cfg *libraryConfig) getLibraries(c *gin.Context) {
	page_str := c.Request.URL.Query().Get("page")
	size_str := c.Request.URL.Query().Get("size")
	city := c.Request.URL.Query().Get("city")

	libraries, err := cfg.queries.GetAllLibraries(c, city)
	if err != nil {
		fmt.Println("error")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}

	page, err := strconv.Atoi(page_str)
	if err != nil || page < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bad_query_params",
		})
		return
	}

	size, err := strconv.Atoi(size_str)
	if err != nil || size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bad_query_params",
		})
		return
	}

	resp_page := []librarydb.Library{}
	elems := 0
	for i, lib := range libraries {
		if i >= (page*size) && i < ((page+1)*size) {
			resp_page = append(resp_page, lib)
			elems++
		}
	}

	type lib struct {
		LibraryUid uuid.UUID `json:"library_uid"`
		Name       string    `json:"name"`
		City       string    `json:"city"`
		Address    string    `json:"address"`
	}

	resp_lib := []lib{}
	for _, rlib := range resp_page {
		resp_lib = append(resp_lib, lib{
			LibraryUid: rlib.LibraryUid,
			Name:       rlib.Name,
			City:       rlib.City,
			Address:    rlib.Address,
		})

	}

	c.JSON(http.StatusOK, gin.H{
		"page":          page,
		"pageSize":      size,
		"totalElements": elems,
		"items":         resp_lib,
	})

}
