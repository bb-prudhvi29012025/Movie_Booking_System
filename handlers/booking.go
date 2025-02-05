// Changes have to be made in this code

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"project/db"
	"project/models"
	"project/utils"

	"github.com/golang-jwt/jwt/v4"
)

func BookSeat(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request at /book-seat")

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		log.Printf("Error retrieving token: %v", err)
		http.Error(w, "Unauthorized: Token not provided", http.StatusUnauthorized)
		return
	}

	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return utils.JwtKey, nil
	})
	if err != nil || !token.Valid {
		log.Printf("Invalid token: %v", err)
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	username := claims.Username
	log.Printf("Authorized user: %s", username)

	var bookingRequest models.BookingRequest
	err = json.NewDecoder(r.Body).Decode(&bookingRequest)
	if err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	theatreID := bookingRequest.TheatreID
	movieID := bookingRequest.MovieID

	var requestedMovieName string
	err = db.DB.QueryRow("SELECT name FROM movies WHERE id = ?", movieID).Scan(&requestedMovieName)
	if err != nil {
		log.Printf("Movie ID %d not found", movieID)
		http.Error(w, "Invalid Movie ID", http.StatusBadRequest)
		return
	}

	var theatreMovieName string
	err = db.DB.QueryRow("SELECT movie FROM theatre WHERE id = ?", theatreID).Scan(&theatreMovieName)
	if err != nil {
		log.Printf("Theatre ID %d not found", theatreID)
		http.Error(w, "Invalid Theatre ID", http.StatusNotFound)
		return
	}

	if requestedMovieName != theatreMovieName {
		log.Printf("Mismatch: Requested movie '%s' is not screening in Theatre ID %d", requestedMovieName, theatreID)
		http.Error(w, "Movie was not screening in this theatre", http.StatusConflict)
		return
	}

	var availableSeats int
	err = db.DB.QueryRow("SELECT seats_vacant FROM theatre WHERE id = ?", theatreID).Scan(&availableSeats)
	if err != nil {
		log.Printf("Error querying seats for Theatre ID %d: %v", theatreID, err)
		http.Error(w, "Theatre not found", http.StatusNotFound)
		return
	}

	if availableSeats <= 0 {
		log.Println("No seats available")
		http.Error(w, "No seats available", http.StatusConflict)
		return
	}

	_, err = db.DB.Exec(
		`UPDATE theatre 
		 SET seats_booked = seats_booked + 1, 
		     seats_vacant = seats_vacant - 1, 
		     updated_by = ?, 
		     updated_on = CURRENT_TIMESTAMP 
		 WHERE id = ?`,
		username, theatreID,
	)
	if err != nil {
		log.Printf("Error updating seats: %v", err)
		http.Error(w, "Failed to book seat", http.StatusInternalServerError)
		return
	}

	response := models.BookingResponse{
		Message: "Seat booked successfully",
		Status:  http.StatusOK,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
