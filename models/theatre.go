package models

import "time"

type Theatre struct {
	ID          int       `json:"id"`
	Theatre_Name       string    `json:"theatre_name"`
	TotalSeats  int       `json:"total_seats"`
	SeatsBooked int       `json:"seats_booked"`
	SeatsVacant int       `json:"seats_vacant"`
	Total_Rooms      int       `json:"total_rooms"`
	CreatedBy   string    `json:"created_by"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedBy   string    `json:"updated_by"`
	UpdatedOn   time.Time `json:"updated_on"`
}
