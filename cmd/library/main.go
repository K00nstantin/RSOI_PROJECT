package main

import (
	"RSOI_PROJECT/internal/librarydb"
	"RSOI_PROJECT/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
	server.POST("/api/v1/libraries/:libraryUid/books/:bookUid/increase", cfg.increaseBookCount)
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
	page--

	size, err := strconv.Atoi(size_str)
	if err != nil || size < 1 || size > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bad_query_params",
		})
		return
	}

	resp_page := []librarydb.Library{}
	totalElems := len(libraries)
	for i, lib := range libraries {
		if i >= (page*size) && i < ((page+1)*size) {
			resp_page = append(resp_page, lib)
		}
	}

	resp_lib := []models.Library{}
	for _, rlib := range resp_page {
		resp_lib = append(resp_lib, models.Library{
			LibraryUid: rlib.LibraryUid,
			Name:       rlib.Name,
			City:       rlib.City,
			Address:    rlib.Address,
		})

	}

	c.JSON(http.StatusOK, gin.H{
		"page":          page,
		"pageSize":      size,
		"totalElements": totalElems,
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
	page--
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

	books := []models.Book{}
	for _, row := range rows {
		books = append(books, models.Book{
			BookUid:        row.BookUid,
			Name:           row.Name_2,
			Author:         row.Author.String,
			Genre:          row.Genre.String,
			Condition:      row.Condition.String,
			AvaliableCount: row.AvailableCount,
		})
	}

	totalElems := len(rows)
	paginated_books := []models.Book{}
	for i, bk := range books {
		if i >= (page*size) && i < ((page+1)*size) {
			if show_all == true {
				paginated_books = append(paginated_books, bk)
			} else {
				if bk.AvaliableCount > 0 {
					paginated_books = append(paginated_books, bk)
				}
			}

		}
	}

	c.JSON(http.StatusOK, gin.H{
		"page":          page + 1,
		"pageSize":      size,
		"totalElements": totalElems,
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

	c.JSON(http.StatusOK, models.Library{
		LibraryUid: library.LibraryUid,
		Name:       library.Name,
		City:       library.City,
		Address:    library.Address,
	})
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

func getCondition(condition string) int {
	switch condition {
	case "EXCELLENT":
		return 3
	case "GOOD":
		return 2
	case "BAD":
		return 1
	default:
		return -1
	}
}

func (cfg *libraryConfig) increaseBookCount(c *gin.Context) {
	bookUidStr := c.Param("bookUid")
	delta := 0
	bookUid, err := uuid.Parse(bookUidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid library id",
			"err":   err,
		})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return
	}
	params := models.CloseReservationParams{}
	if err := json.Unmarshal(body, &params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to unmarshall",
			"err":   err,
		})
		return
	}
	book, err := cfg.queries.GetBook(c, bookUid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to find book",
			"err":   err,
		})
		return
	}
	pre_condition := getCondition(book.Condition.String)
	post_condition := getCondition(params.Condition)
	if post_condition < pre_condition {
		delta -= 10
	}
	cond := sql.NullString{
		String: params.Condition,
		Valid:  true,
	}
	condition_params := librarydb.ChangeBookConditionParams{
		Condition: cond,
		BookUid:   bookUid,
	}
	err = cfg.queries.ChangeBookCondition(c, condition_params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update condition",
			"err":   err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"delta": delta,
	})
}
