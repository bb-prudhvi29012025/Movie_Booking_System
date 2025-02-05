package models

import "time"

type Room struct {
	ID          int       `json:"id"`
	RoomName    string    `json:"room_name"`
	MovieName   string    `json:"movie_name"`
	SeatsVacant int       `json:"seats_vacant"`
	SeatsBooked int       `json:"seats_booked"`
	TheatreID   int       `json:"theatre_id"`
	CreatedBy   string    `json:"created_by"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedBy   string    `json:"updated_by"`
	UpdatedOn   time.Time `json:"updated_on"`
}
