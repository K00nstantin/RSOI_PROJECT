package main

import (
	"RSOI_PROJECT/internal/auth"
	"RSOI_PROJECT/internal/kafka"
	"RSOI_PROJECT/internal/models"
	"RSOI_PROJECT/internal/reservationdb"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type gatewayConfig struct {
	client                http.Client
	libraryServiceURL     string
	reservationServiceURL string
	ratingServiceURL      string
	identityServiceURL    string
	statisticsServiceURL  string
	publicKey             interface{}
	kafkaProducer         *kafka.Producer
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading .env")
	}

	producer, err := kafka.NewProducer([]string{os.Getenv("KAFKA_BROKERS")}, "library-events")
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	myClient := http.Client{
		Timeout: 10 * time.Second,
	}
	auth_cfg := auth.NewConfig()
	cfg := gatewayConfig{
		client:                myClient,
		libraryServiceURL:     os.Getenv("LIBRARY_SERVICE_URL"),
		reservationServiceURL: os.Getenv("RESERVATION_SERVICE_URL"),
		ratingServiceURL:      os.Getenv("RATING_SERVICE_URL"),
		identityServiceURL:    os.Getenv("IDENTITY_SERVICE_URL"),
		statisticsServiceURL:  os.Getenv("STATISTICS_SERVICE_URL"),
		kafkaProducer:         producer,
	}
	if err = auth_cfg.LoadJWKS(cfg.identityServiceURL); err != nil {
		log.Fatalf("Failed to load JWKS: %v", err)
	}
	r := gin.Default()
	r.Use(auth_cfg.AuthMiddleware())
	r.GET("/api/v1/libraries", cfg.getLibrariesHandler)
	r.GET("/api/v1/libraries/:libraryUid/books", cfg.getLibraryBooksHandler)
	r.GET("/api/v1/reservations", cfg.getReservationsHandler)
	r.POST("/api/v1/reservations", cfg.createReservationHandler)
	r.POST("/api/v1/reservations/:reservationUid/return", cfg.returnBookHandler)
	r.GET("/api/v1/rating", cfg.getRatingHandler)
	r.POST("/api/v1/createUser", cfg.createUserHandler)
	r.GET("/api/v1/stats/report", cfg.statsReportHandler)
	r.GET("/manage/health", healthCheck)

	r.Run(":8080")
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "up",
	})
}

func (cfg *gatewayConfig) getLibrariesHandler(c *gin.Context) {
	query_params := c.Request.URL.Query().Encode()
	city := c.Request.URL.Query().Get("city")
	page := c.Query("page")
	if city == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Missing required query parameter: city",
			"errors": []gin.H{
				{
					"field": "city",
					"error": "city is required",
				},
			},
		})
		return
	}

	request_string := cfg.libraryServiceURL + "/api/v1/libraries" + "?" + query_params

	req, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	req.Header.Set("Authorization", c.GetHeader("Authorization"))
	resp, err := cfg.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}

	username := c.GetString("username")
	if username != "" {
		event := models.NewEvent(username, "search_libraries",
			fmt.Sprintf("city=%s, page=%d", city, page))
		if err := cfg.kafkaProducer.SendEvent(event); err != nil {
			log.Printf("Failed to send event: %v", err)
		}
	}

	c.Data(http.StatusOK, "application/json", body)
}

func (cfg *gatewayConfig) getLibraryBooksHandler(c *gin.Context) {
	libraryUid := c.Param("libraryUid")
	other_params := c.Request.URL.RawQuery
	request_string := cfg.libraryServiceURL + "/api/v1/libraries/" + libraryUid + "/books?" + other_params

	request, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
		})
		return
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	resp, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to send request",
			"err":   err,
		})
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return
	}

	username := c.GetString("username")
	if username != "" {
		event := models.NewEvent(username, "view_books",
			fmt.Sprintf("libraryUid=%s", libraryUid))
		if err := cfg.kafkaProducer.SendEvent(event); err != nil {
			log.Printf("Failed to send event: %v", err)
		}
	}

	c.Data(http.StatusOK, "application/json", body)
}

func (cfg *gatewayConfig) getReservationsHandler(c *gin.Context) {
	request_string := cfg.reservationServiceURL + "/api/v1/reservations"
	req, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
			"err":   err,
		})
		return
	}
	req.Header.Set("Authorization", c.GetHeader("Authorization"))
	resp, err := cfg.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "failed to perform request",
			"err":   err,
		})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error reading body",
			"err":   err,
		})
		return
	}

	type wrapper struct {
		Reservations []models.Reservation `json:"reservations"`
	}
	wrapped := wrapper{}

	err = json.Unmarshal(body, &wrapped)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error decoding",
			"err":   err,
		})
		return
	}

	final_resp := []models.GetReservationResponse{}
	for _, reserv := range wrapped.Reservations {
		lib, err := cfg.getLibraryByUUID(c, reserv.LibraryUid)

		bk_dto, err := cfg.getBookByUUID(c, reserv.BookUid, reserv.LibraryUid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}

		final_resp = append(final_resp, models.GetReservationResponse{
			ReservationUid: reserv.ReservationUid,
			Status:         reserv.Status,
			StartDate:      reserv.StartDate,
			TillDate:       reserv.TillDate,
			Library:        lib,
			Book:           bk_dto,
		})
	}
	username := c.GetString("username")
	if username != "" {
		event := models.NewEvent(username, "view_reservations", "")
		cfg.kafkaProducer.SendEvent(event)
	}
	c.JSON(http.StatusOK, final_resp)
}

func (cfg *gatewayConfig) getLibraryByUUID(c *gin.Context, uuid uuid.UUID) (models.Library, error) {
	request_string := cfg.libraryServiceURL + "/api/v1/libraries/" + uuid.String()
	request, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while creating getlibrarybyuuid request",
			"err":   err,
		})
		return models.Library{}, err
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while making getlibrarybyuuid request",
			"err":   err,
		})
		return models.Library{}, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while reading body",
			"err":   err,
		})
		return models.Library{}, err
	}
	lib := models.Library{}
	if err = json.Unmarshal(body, &lib); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while reading body",
			"err":   err,
		})
		return models.Library{}, err
	}

	return lib, nil
}

func (cfg *gatewayConfig) getBookByUUID(c *gin.Context, book_uuid uuid.UUID, libraryuuid uuid.UUID) (models.BookDTO, error) {
	request_string := cfg.libraryServiceURL + "/api/v1/libraries/" + libraryuuid.String() + "/books/" + book_uuid.String()
	request, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while creating getlibrarybyuuid request",
			"err":   err,
		})
		return models.BookDTO{}, err
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while making getbookbyuuid request",
			"err":   err,
		})
		return models.BookDTO{}, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while reading body",
			"err":   err,
		})
		return models.BookDTO{}, err
	}

	bk := models.SQLbook{}
	if err = json.Unmarshal(body, &bk); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while unmarrshalling",
			"err":   err,
		})
		return models.BookDTO{}, err
	}
	dto := models.BookDTO{
		BookUid: bk.BookUid,
		Name:    bk.Name,
		Author:  bk.Author.String,
		Genre:   bk.Genre.String,
	}
	return dto, nil
}

func (cfg *gatewayConfig) createReservationHandler(c *gin.Context) {
	books_count, err := cfg.getBooksCount(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get books count",
			"err":   err,
		})
		return
	}

	stars_count, err := cfg.getRating(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get stars count",
			"err":   err,
		})
		return
	}

	if books_count >= stars_count.Stars {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "too many books rented",
			"books": books_count,
			"stars": stars_count,
		})
		return
	}

	req_body := models.CreateReservationBody{}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err = json.Unmarshal(body, &req_body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to unmarshall",
			"err":   err,
		})
		return
	}

	resp_body, err := cfg.createReservation(c, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create reservation",
			"err":   err,
		})
		return
	}

	type reservationDTO struct {
		Reservation reservationdb.Reservation
	}
	resp := reservationDTO{}
	if err = json.Unmarshal(resp_body, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error while unmarshalling",
			"err":   err,
		})
		return
	}

	book, err := cfg.getBookByUUID(c, req_body.BookUid, req_body.LibraryUid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "error getting book by id",
			"err":   err,
		})
		return
	}
	fmt.Println(req_body.LibraryUid)
	library, err := cfg.getLibraryByUUID(c, req_body.LibraryUid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "error getting book by id",
			"err":   err,
		})
		return
	}

	rating, err := cfg.getRating(c)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "error getting rating",
			"err":   err,
		})
		return
	}
	err = cfg.decreaseBookCount(c, req_body.BookUid, req_body.LibraryUid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to decrease book count",
			"err":   err,
		})
		return
	}
	response := models.ReservationDTO{
		ReservationUid: resp.Reservation.ReservationUid,
		Status:         resp.Reservation.Status,
		StartDate:      models.Date(resp.Reservation.StartDate),
		TillDate:       models.Date(resp.Reservation.TillDate),
		Book:           book,
		Library:        library,
		Rating:         rating,
	}

	username := c.GetString("username")
	if username != "" {
		event := models.NewEvent(username, "book_rent",
			fmt.Sprintf("bookUid=%s, libraryUid=%s", req_body.BookUid, req_body.LibraryUid))
		cfg.kafkaProducer.SendEvent(event)
	}
	c.JSON(http.StatusOK, response)

}

func (cfg *gatewayConfig) getBooksCount(c *gin.Context) (int, error) {
	request_string := cfg.reservationServiceURL + "/api/v1/reservations/active/count"
	request, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
			"err":   err,
		})
		return -1, err
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "failed to make a request",
			"err":   err,
		})
		return -1, err
	}
	response_body, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return -1, err
	}

	type count struct {
		Count int `json:"count"`
	}
	book_number := count{}

	if err = json.Unmarshal(response_body, &book_number); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to unmarshall",
			"err":   err,
		})
		return -1, err
	}
	return book_number.Count, nil
}

func (cfg *gatewayConfig) getRating(c *gin.Context) (models.Rating, error) {
	request_string := cfg.ratingServiceURL + "/api/v1/rating"
	request, err := http.NewRequest("GET", request_string, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
			"err":   err,
		})
		return models.Rating{}, err
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to make a request",
			"err":   err,
		})
		return models.Rating{}, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return models.Rating{}, err
	}

	stars := models.Rating{}

	if err = json.Unmarshal(body, &stars); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to unmarshall",
			"err":   err,
		})
		return models.Rating{}, err
	}
	return stars, nil
}

func (cfg *gatewayConfig) createReservation(c *gin.Context, body io.Reader) ([]byte, error) {
	request_string := cfg.reservationServiceURL + "/api/v1/reservations"
	request, err := http.NewRequest("POST", request_string, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
			"err":   err,
		})
		return nil, err
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to make a request",
			"err":   err,
		})
		return nil, err
	}
	resp_body, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return nil, err
	}

	return resp_body, nil
}

func (cfg *gatewayConfig) decreaseBookCount(c *gin.Context, book_uuid uuid.UUID, libraryuuid uuid.UUID) error {
	request_string := cfg.libraryServiceURL + "/api/v1/libraries/" + libraryuuid.String() + "/books/" + book_uuid.String() + "/decrease"
	request, err := http.NewRequest("POST", request_string, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("could not decrease book count")
	}
	return nil
}

func (cfg *gatewayConfig) returnBookHandler(c *gin.Context) {
	username := c.GetString("username")
	reservationUid_str := c.Param("reservationUid")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return
	}
	body_1 := bytes.NewReader(body)
	body_2 := bytes.NewReader(body)

	err, bookUid, libraryUid, del := cfg.closeReservation(c, reservationUid_str, body_1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to close reservation",
			"err":   err,
		})
		return
	}

	err, delta := cfg.returnBook(c, libraryUid, bookUid, body_2, del)
	if delta == 0 {
		delta++
	}

	update_request_string := cfg.ratingServiceURL + "/api/v1/rating"
	update_body, err := json.Marshal(gin.H{
		"stars":    delta,
		"username": username,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to marshall",
			"err":   err,
		})
		return
	}

	update_request, err := http.NewRequest("PUT", update_request_string, bytes.NewReader(update_body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create request",
			"err":   err,
		})
		return
	}
	update_request.Header.Set("Authorization", c.GetHeader("Authorization"))
	_, err = cfg.client.Do(update_request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to make a request",
			"err":   err,
		})
		return
	}

	if username != "" {
		event := models.NewEvent(username, "book_return",
			fmt.Sprintf("reservationUid=%s", reservationUid_str))
		cfg.kafkaProducer.SendEvent(event)
	}
	c.JSON(http.StatusNoContent, nil)

}

func (cfg *gatewayConfig) closeReservation(c *gin.Context, reservaton_uuid_str string, io_body io.Reader) (error, uuid.UUID, uuid.UUID, int) {
	request_str := cfg.reservationServiceURL + "/api/v1/reservations/" + reservaton_uuid_str + "/return"
	request, err := http.NewRequest("POST", request_str, io_body)
	if err != nil {
		return err, uuid.Nil, uuid.Nil, 0
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil {
		return err, uuid.Nil, uuid.Nil, 0
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err, uuid.Nil, uuid.Nil, 0
	}

	resp := models.CloseReservationResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err, uuid.Nil, uuid.Nil, 0
	}
	return nil, resp.BookUid, resp.LibraryUid, resp.Delta

}

func (cfg *gatewayConfig) returnBook(c *gin.Context, libraryUid, bookUid uuid.UUID, io_body io.Reader, del int) (error, int) {
	request_str := cfg.libraryServiceURL + "/api/v1/libraries/" + libraryUid.String() + "/books/" + bookUid.String() + "/increase"
	request, err := http.NewRequest("POST", request_str, io_body)
	if err != nil {
		return err, 0
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil {
		return err, 0
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err, 0
	}
	type delta struct {
		Delta int
	}
	params := delta{}
	if err = json.Unmarshal(body, &params); err != nil {
		return err, 0
	}
	return nil, params.Delta + del
}

func (cfg *gatewayConfig) getRatingHandler(c *gin.Context) {
	request_str := cfg.ratingServiceURL + "/api/v1/rating"
	request, err := http.NewRequest("GET", request_str, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get rating",
			"err":   err,
		})
		return
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	response, err := cfg.client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to make a request",
			"err":   err,
		})
		return
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body",
			"err":   err,
		})
		return
	}
	username := c.GetString("username")
	if username != "" {
		event := models.NewEvent(username, "view_rating", "")
		cfg.kafkaProducer.SendEvent(event)
	}
	c.Data(http.StatusOK, "application/json", body)
}

func (cfg *gatewayConfig) createUserHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	targetURL := cfg.identityServiceURL + "/api/v1/createUser"
	httpReq, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	httpReq.Header = c.Request.Header.Clone()
	httpReq.Header.Set("Authorization", c.GetHeader("Authorization"))

	resp, err := cfg.client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	if resp.StatusCode == http.StatusCreated {
		go cfg.initRatingForUser(req.Username, c.GetHeader("Authorization"))
	}

	c.Data(resp.StatusCode, "application/json", respBody)
}

func (cfg *gatewayConfig) initRatingForUser(username, authHeader string) {
	initReq := struct {
		Username string `json:"username"`
		Stars    int32  `json:"stars"`
	}{
		Username: username,
		Stars:    25,
	}
	body, _ := json.Marshal(initReq)
	url := cfg.ratingServiceURL + "/api/v1/rating/init"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{Timeout: 5 * time.Second}
	if resp, err := client.Do(req); err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			log.Printf("Rating init failed with status %d for user %s", resp.StatusCode, username)
		}
	} else {
		log.Printf("Failed to init rating for user %s: %v", username, err)
	}
}

func (cfg *gatewayConfig) statsReportHandler(c *gin.Context) {
	targetURL := cfg.statisticsServiceURL + "/api/v1/stats/report"

	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	req.Header = c.Request.Header.Clone()
	req.Header.Set("Authorization", c.GetHeader("Authorization"))

	resp, err := cfg.client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach statistics service"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(resp.StatusCode, "application/json", body)
}
