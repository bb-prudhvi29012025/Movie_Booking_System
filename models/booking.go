package models

type BookingRequest struct {
	MovieID   int `json:"movie_id"`
	TheatreID int `json:"theatre_id"`
}

type BookingResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}
