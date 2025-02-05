package handlers

import (
	"fmt"
	"encoding/json"
	"net/http"
	"project/db"
	"project/models"
	"time"
)

func authenticateSystem(w http.ResponseWriter, r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok || username != "System" || password != "Test@123" {	
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func InsertTheatre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var theatre models.Theatre
	err := json.NewDecoder(r.Body).Decode(&theatre)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if theatre.TotalSeats <= 10 {
		http.Error(w, "Total seats must be greater than 10", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO theatre (theatre_name, total_seats, seats_booked, seats_vacant, total_rooms, created_by, created_on)
		VALUES (?, ?, 0, ?, ?, 'System', ?)`,
		theatre.Theatre_Name, theatre.TotalSeats, theatre.TotalSeats, theatre.Total_Rooms, time.Now(),
	)
	if err != nil {
		http.Error(w, "Failed to insert theatre", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Theatre added successfully"})
}

func UpdateTheatre(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    if !authenticateSystem(w, r) {
        return
    }

    var updateData map[string]interface{}
    err := json.NewDecoder(r.Body).Decode(&updateData)
    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    id, ok := updateData["id"].(float64)
    if !ok {
        http.Error(w, "ID is required", http.StatusBadRequest)
        return
    }

    var seatsBooked int
    err = db.DB.QueryRow("SELECT seats_booked FROM theatre WHERE id = ?", int(id)).Scan(&seatsBooked)
    if err != nil {
        http.Error(w, "Theatre not found", http.StatusNotFound)
        return
    }

    if totalSeatsVal, ok := updateData["total_seats"].(float64); ok && totalSeatsVal < float64(seatsBooked) {
        http.Error(w, "Total seats cannot be less than booked seats", http.StatusBadRequest)
        return
    }

    query := "UPDATE theatre SET "
    params := []interface{}{}
    paramCount := 0

    for key, value := range updateData {
        if key == "id" {
            continue // Skip the "id" field
        }

        query += fmt.Sprintf("%s = ?, ", key)
        params = append(params, value)
        paramCount++
    }

    if paramCount == 0 {
        http.Error(w, "No attributes to update", http.StatusBadRequest)
        return
    }

    query = query[:len(query)-2]

    query += ", updated_by = 'System', updated_on = ? WHERE id = ?"
    params = append(params, time.Now(), int(id))

    _, err = db.DB.Exec(query, params...)
    if err != nil {
        http.Error(w, "Failed to update theatre", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{"message": "Theatre updated successfully"})
}

func DeleteTheatre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var req struct {
		ID int `json:"theatre_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var bookedSeats int
	err = db.DB.QueryRow("SELECT seats_booked FROM theatre WHERE id = ?", req.ID).Scan(&bookedSeats)
	if err != nil {
		http.Error(w, "Theatre not found", http.StatusNotFound)
		return
	}

	if bookedSeats > 0 {
		http.Error(w, "Cannot delete theatre with booked seats", http.StatusConflict)
		return
	}

	_, err = db.DB.Exec("DELETE FROM theatre WHERE id = ?", req.ID)
	if err != nil {
		http.Error(w, "Failed to delete theatre", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Theatre deleted successfully"})
}

func GetTheatreByTheatreName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	theatreName := r.URL.Query().Get("theatre_name")
	if theatreName == "" {
		http.Error(w, "Theatre name is required", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query("SELECT id, theatre_name, total_seats, seats_booked, seats_vacant, total_rooms FROM theatre WHERE theatre_name = ?", theatreName)
	if err != nil {
		http.Error(w, "Failed to retrieve theatres", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var theatres []struct {
		ID          int    `json:"id"`
		Theatre_Name string `json:"theatre_name"`
		TotalSeats  int    `json:"total_seats"`
		SeatsBooked int    `json:"seats_booked"`
		SeatsVacant int    `json:"seats_vacant"`
		RoomNo      int    `json:"total_rooms"`
	}

	for rows.Next() {
		var theatre struct {
			ID          int    `json:"id"`
			Theatre_Name string `json:"theatre_name"`
			TotalSeats  int    `json:"total_seats"`
			SeatsBooked int    `json:"seats_booked"`
			SeatsVacant int    `json:"seats_vacant"`
			RoomNo      int    `json:"total_rooms"`
		}
		err := rows.Scan(&theatre.ID, &theatre.Theatre_Name, &theatre.TotalSeats, &theatre.SeatsBooked, &theatre.SeatsVacant, &theatre.RoomNo)
		if err != nil {
			http.Error(w, "Failed to read theatre data", http.StatusInternalServerError)
			return
		}
		theatres = append(theatres, theatre)
	}

	if len(theatres) == 0 {
		http.Error(w, "No theatres found with the given theatre name", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(theatres, "", "  ")
	if err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
	w.Write([]byte("\n"))
}


func InsertMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var movie models.Movie
	err := json.NewDecoder(r.Body).Decode(&movie)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO movie (movie_name, description)
		VALUES (?, ?)`,
		movie.Movie_Name, movie.Description,
	)
	if err != nil {
		http.Error(w, "Failed to insert movie", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Movie added successfully"})
}

func UpdateMovie(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    if !authenticateSystem(w, r) {
        return
    }

    var updateData map[string]interface{}
    err := json.NewDecoder(r.Body).Decode(&updateData)
    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    id, ok := updateData["id"].(float64)
    if !ok {
        http.Error(w, "ID is required", http.StatusBadRequest)
        return
    }

    var movieName, description string
    err = db.DB.QueryRow("SELECT movie_name, description FROM movie WHERE id = ?", int(id)).Scan(&movieName, &description)
    if err != nil {
        http.Error(w, "Movie not found", http.StatusNotFound)
        return
    }

    query := "UPDATE movie SET "
    params := []interface{}{}
    paramCount := 0

    for key, value := range updateData {
        if key == "id" {
            continue 
        }

        query += fmt.Sprintf("%s = ?, ", key)
        params = append(params, value)
        paramCount++
    }

    if paramCount == 0 {
        http.Error(w, "No attributes to update", http.StatusBadRequest)
        return
    }

    query = query[:len(query)-2]

    query += " WHERE id = ?"
    params = append(params, int(id))

    _, err = db.DB.Exec(query, params...)
    if err != nil {
        http.Error(w, "Failed to update movie", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{"message": "Movie updated successfully"})
}

func DeleteMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var req struct {
		ID int `json:"movie_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM movie WHERE id = ?", req.ID)
	if err != nil {
		http.Error(w, "Failed to delete movie", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Movie deleted successfully"})
}

func GetMovieByMovieName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	movieName := r.URL.Query().Get("movie_name")
	if movieName == "" {
		http.Error(w, "Movie name is required", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query("SELECT id, movie_name, description FROM movie WHERE movie_name = ?", movieName)
	if err != nil {
		http.Error(w, "Failed to retrieve movies", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var movies []struct {
		ID          int    `json:"id"`
		Movie_Name  string `json:"movie_name"`
		Description string `json:"description"`
	}

	for rows.Next() {
		var movie struct {
			ID          int    `json:"id"`
			Movie_Name  string `json:"movie_name"`
			Description string `json:"description"`
		}
		err := rows.Scan(&movie.ID, &movie.Movie_Name, &movie.Description)
		if err != nil {
			http.Error(w, "Failed to read movie data", http.StatusInternalServerError)
			return
		}
		movies = append(movies, movie)
	}

	if len(movies) == 0 {
		http.Error(w, "No movies found with the given movie name", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
	w.Write([]byte("\n"))
}
