package models

type BookingRequest struct {
    TheatreName string `json:"theatre_name"`
    MovieName   string `json:"movie_name"`
    RoomName    string `json:"room_name,omitempty"`
    NoOfSeats   int    `json:"no_of_seats"`
}

type BookingResponse struct {
    Message string `json:"message"`
    Status  int    `json:"status"`
}
