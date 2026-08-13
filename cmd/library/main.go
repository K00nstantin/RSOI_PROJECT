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
	server.GET("/api/v1/libraries/:libraryUid", cfg.getLibrary)
	server.GET("/api/v1/libraries/:libraryUid/books", cfg.getLibraryBooks)
	server.GET("/api/v1/libraries/:libraryUid/books/:bookUid", cfg.getLibraryBook)
	server.POST("/api/v1/libraries/:libraryUid/books/:bookUid/decrease", cfg.decreaseBookCount)
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

func (cfg *libraryConfig) getLibraryBooks(c *gin.Context) {
	libraryUid_string := c.Param("libraryUid")
	libraryUid, err := uuid.Parse(libraryUid_string)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id",
			"err":   err,
		})
		return
	}
	page_str := c.Request.URL.Query().Get("page")
	page, err := strconv.Atoi(page_str)
	if err != nil || page < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "wrong page parameter",
		})
		return
	}
	size_str := c.Request.URL.Query().Get("size")
	size, err := strconv.Atoi(size_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "wrong size parameter",
		})
		return
	}

	show_all_str := c.Request.URL.Query().Get("showAll")
	show_all := false
	if show_all_str == "true" {
		show_all = true
	}

	rows, err := cfg.queries.GetLibraryBooks(c, libraryUid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}

	type book struct {
		BookUid        uuid.UUID `json:"bookUid"`
		Name           string    `json:"name"`
		Author         string    `json:"author"`
		Genre          string    `json:"genre"`
		Condition      string    `json:"condition"`
		AvaliableCount int32     `json:"avaliableCount"`
	}

	books := []book{}
	for _, row := range rows {
		books = append(books, book{
			BookUid:        row.BookUid,
			Name:           row.Name_2,
			Author:         row.Author.String,
			Genre:          row.Genre.String,
			Condition:      row.Condition.String,
			AvaliableCount: row.AvailableCount,
		})
	}

	elems := 0
	paginated_books := []book{}
	for i, bk := range books {
		if i >= (page*size) && i < ((page+1)*size) {
			if show_all == true {
				paginated_books = append(paginated_books, bk)
				elems++
			} else {
				if bk.AvaliableCount > 0 {
					paginated_books = append(paginated_books, bk)
					elems++
				}
			}

		}
	}

	c.JSON(http.StatusOK, gin.H{
		"page":          page,
		"size":          size,
		"totalelements": elems,
		"items":         paginated_books,
	})
}

func (cfg *libraryConfig) getLibrary(c *gin.Context) {
	libraryUid_str := c.Param("libraryUid")
	libraryUid, err := uuid.Parse(libraryUid_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id",
			"err":   err,
		})
		return
	}
	library, err := cfg.queries.GetLibrary(c, libraryUid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed db query",
			"err":   err,
		})
		return
	}
	c.JSON(http.StatusOK, library)
}

func (cfg *libraryConfig) getLibraryBook(c *gin.Context) {
	book_id_str := c.Param("bookUid")
	fmt.Println(book_id_str)
	book_id, err := uuid.Parse(book_id_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
			"err":   err,
		})
		return
	}
	book, err := cfg.queries.GetBook(c, book_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while getting book info",
			"err":   err,
		})
		return
	}
	c.JSON(http.StatusOK, book)

}

func (cfg *libraryConfig) decreaseBookCount(c *gin.Context) {
	libraryUid_str := c.Param("libraryUid")
	if libraryUid_str == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": "invalid library id",
		})
		return
	}
	libraryUid, err := uuid.Parse(libraryUid_str)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": "invalid library id",
		})
		return
	}
	bookUid_str := c.Param("bookUid")
	if bookUid_str == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": "invalid book id",
		})
		return
	}
	bookUid, err := uuid.Parse(bookUid_str)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": "invalid book id",
		})
		return
	}
	params := librarydb.DecreaseBookCountParams{
		LibraryUid: libraryUid,
		BookUid:    bookUid,
	}
	_, err = cfg.queries.DecreaseBookCount(c, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "wrong parameters",
			"err":   err,
		})
		return
	}
	c.JSON(http.StatusOK, nil)
}
