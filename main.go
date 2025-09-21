package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"googlemaps.github.io/maps"
)

// Response structures
type ValidationResponse struct {
	Valid       bool     `json:"valid"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions,omitempty"`
	Details     *Details `json:"details,omitempty"`
}

type Details struct {
	PinCode          string `json:"pin_code"`
	City             string `json:"city"`
	State            string `json:"state"`
	Country          string `json:"country"`
	FormattedAddress string `json:"formatted_address"`
}

type LandmarksResponse struct {
	Success   bool       `json:"success"`
	Message   string     `json:"message"`
	Landmarks []Landmark `json:"landmarks"`
	Location  Location   `json:"location"`
}

type Landmark struct {
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Distance float64  `json:"distance"`
	PlaceID  string   `json:"place_id"`
	Types    []string `json:"types"`
	Location Location `json:"location"`
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type ValidatePinCodeRequest struct {
	PinCode string `json:"pin_code"`
	City    string `json:"city"`
}

type GetLandmarksRequest struct {
	PinCode string  `json:"pin_code"`
	City    string  `json:"city"`
	Radius  float64 `json:"radius,omitempty"` // in meters, default 1000
}

// Service structure
type LocationService struct {
	mapsClient *maps.Client
}

// NewLocationService creates a new location service instance
func NewLocationService(apiKey string) (*LocationService, error) {
	client, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create maps client: %v", err)
	}
	return &LocationService{mapsClient: client}, nil
}

// ValidatePinCodeWithCity validates if the PIN code matches the city
func (s *LocationService) ValidatePinCodeWithCity(ctx context.Context, pinCode, city string) (*ValidationResponse, error) {
	// Clean inputs
	pinCode = strings.TrimSpace(pinCode)
	city = strings.ToLower(strings.TrimSpace(city))

	if pinCode == "" || city == "" {
		return &ValidationResponse{
			Valid:   false,
			Message: "PIN code and city are required",
		}, nil
	}

	// Geocode the PIN code to get location details
	geocodeReq := &maps.GeocodingRequest{
		Address: pinCode,
		Components: map[maps.Component]string{
			maps.ComponentPostalCode: pinCode,
		},
	}

	results, err := s.mapsClient.Geocode(ctx, geocodeReq)
	if err != nil {
		return nil, fmt.Errorf("geocoding failed: %v", err)
	}

	if len(results) == 0 {
		return &ValidationResponse{
			Valid:   false,
			Message: "Invalid PIN code: No location found",
		}, nil
	}

	// Extract city from the geocoding results
	var foundCity, foundState, foundCountry string
	var formattedAddress string

	for _, result := range results {
		formattedAddress = result.FormattedAddress
		for _, component := range result.AddressComponents {
			for _, typ := range component.Types {
				switch typ {
				case "locality", "administrative_area_level_2":
					if foundCity == "" {
						foundCity = strings.ToLower(component.LongName)
					}
				case "administrative_area_level_1":
					foundState = component.LongName
				case "country":
					foundCountry = component.LongName
				}
			}
		}

		// Check if the provided city matches
		if strings.Contains(foundCity, city) || strings.Contains(city, foundCity) {
			return &ValidationResponse{
				Valid:   true,
				Message: "PIN code and city match successfully",
				Details: &Details{
					PinCode:          pinCode,
					City:             foundCity,
					State:            foundState,
					Country:          foundCountry,
					FormattedAddress: formattedAddress,
				},
			}, nil
		}
	}

	// If no match found, suggest correct city
	suggestions := []string{}
	if foundCity != "" {
		suggestions = append(suggestions, foundCity)
	}

	return &ValidationResponse{
		Valid:       false,
		Message:     fmt.Sprintf("PIN code %s does not belong to %s", pinCode, city),
		Suggestions: suggestions,
		Details: &Details{
			PinCode:          pinCode,
			City:             foundCity,
			State:            foundState,
			Country:          foundCountry,
			FormattedAddress: formattedAddress,
		},
	}, nil
}

// GetNearbyLandmarks fetches nearby landmarks for a given PIN code
func (s *LocationService) GetNearbyLandmarks(ctx context.Context, pinCode, city string, radius float64) (*LandmarksResponse, error) {
	// First validate the PIN code
	validation, err := s.ValidatePinCodeWithCity(ctx, pinCode, city)
	if err != nil {
		return nil, err
	}

	if !validation.Valid {
		return &LandmarksResponse{
			Success: false,
			Message: validation.Message,
		}, nil
	}

	// Geocode to get exact coordinates
	geocodeReq := &maps.GeocodingRequest{
		Address: fmt.Sprintf("%s, %s", pinCode, city),
	}

	geocodeResults, err := s.mapsClient.Geocode(ctx, geocodeReq)
	if err != nil {
		return nil, fmt.Errorf("geocoding failed: %v", err)
	}

	if len(geocodeResults) == 0 {
		return &LandmarksResponse{
			Success: false,
			Message: "Could not find location coordinates",
		}, nil
	}

	location := geocodeResults[0].Geometry.Location

	// Default radius
	if radius == 0 {
		radius = 1000 // 1km default
	}

	// Search for nearby landmarks
	nearbyReq := &maps.NearbySearchRequest{
		Location: &location,
		Radius:   uint(radius),
		Type:     maps.PlaceType("point_of_interest"),
	}

	nearbyResults, err := s.mapsClient.NearbySearch(ctx, nearbyReq)
	if err != nil {
		return nil, fmt.Errorf("nearby search failed: %v", err)
	}

	// Process and select top 3 landmarks
	landmarks := []Landmark{}
	for i, place := range nearbyResults.Results {
		if i >= 3 {
			break
		}

		// Calculate distance
		distance := calculateDistance(
			location.Lat, location.Lng,
			place.Geometry.Location.Lat, place.Geometry.Location.Lng,
		)

		landmark := Landmark{
			Name:     place.Name,
			Address:  place.Vicinity,
			Distance: distance,
			PlaceID:  place.PlaceID,
			Types:    place.Types,
			Location: Location{
				Lat: place.Geometry.Location.Lat,
				Lng: place.Geometry.Location.Lng,
			},
		}
		landmarks = append(landmarks, landmark)
	}

	return &LandmarksResponse{
		Success:   true,
		Message:   "Landmarks fetched successfully",
		Landmarks: landmarks,
		Location: Location{
			Lat: location.Lat,
			Lng: location.Lng,
		},
	}, nil
}

// calculateDistance calculates distance between two coordinates in meters
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // meters

	lat1Rad := lat1 * (3.14159265359 / 180)
	lat2Rad := lat2 * (3.14159265359 / 180)
	deltaLat := (lat2 - lat1) * (3.14159265359 / 180)
	deltaLon := (lon2 - lon1) * (3.14159265359 / 180)

	a := (deltaLat/2)*(deltaLat/2) +
		(lat1Rad)*(lat2Rad)*(deltaLon/2)*(deltaLon/2)
	c := 2 * a

	return earthRadius * c
}

// HTTP Handlers
func (s *LocationService) handleValidatePinCode(w http.ResponseWriter, r *http.Request) {
	var req ValidatePinCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := s.ValidatePinCodeWithCity(ctx, req.PinCode, req.City)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *LocationService) handleGetLandmarks(w http.ResponseWriter, r *http.Request) {
	var req GetLandmarksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := s.GetNearbyLandmarks(ctx, req.PinCode, req.City, req.Radius)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get landmarks: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Middleware for logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Get API key from environment
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_MAPS_API_KEY environment variable is required")
	}

	// Initialize service
	service, err := NewLocationService(apiKey)
	if err != nil {
		log.Fatalf("Failed to initialize location service: %v", err)
	}

	// Setup routes
	router := mux.NewRouter()

	// === API endpoints ===
	router.HandleFunc("/api/validate-pincode", service.handleValidatePinCode).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/get-landmarks", service.handleGetLandmarks).Methods("POST", "OPTIONS")

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	// === Serve static frontend ===
	// Put index.html and assets inside ./static/
	fs := http.FileServer(http.Dir("./static"))
	router.PathPrefix("/").Handler(fs)

	// Apply middleware
	handler := loggingMiddleware(corsMiddleware(router))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	log.Printf("Endpoints:")
	log.Printf("  POST /api/validate-pincode - Validate PIN code with city")
	log.Printf("  POST /api/get-landmarks - Get nearby landmarks")
	log.Printf("  GET  /health - Health check")
	log.Printf("  GET  /        - Frontend UI")

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
