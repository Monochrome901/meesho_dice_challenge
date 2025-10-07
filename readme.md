# Smart Address Validator & Landmark Finder

A Go web application that validates Indian PIN codes, matches them with cities, and finds nearby landmarks using the Google Maps API.

## Features

- PIN code and city validation
- Street address geocoding
- Nearby landmark discovery with smart ranking
- Interactive web UI
- RESTful API endpoints

## Tech Stack

- Backend: Go 1.25+
- Frontend: HTML5, CSS3, JavaScript
- APIs: Google Maps Geocoding, Places
- Dependencies:
  - `github.com/gorilla/mux`: Router
  - `github.com/joho/godotenv`: Environment config
  - `googlemaps.github.io/maps`: Google Maps client

## Setup

1. Clone the repository:
```sh
git clone <repository-url>
cd <project-directory>
```

2. Create a `.env` file in the project root:
```sh
GOOGLE_MAPS_API_KEY=your_api_key_here
PORT=8080
```

3. Install dependencies:
```sh
go mod download
```

4. Run the server:
```sh
go run main.go
```

The application will be available at `http://localhost:8080`

## API Endpoints

### 1. Validate PIN Code
```http
POST /api/validate-pincode
Content-Type: application/json

{
    "pin_code": "208001",
    "city": "Kanpur"
}
```

### 2. Get Nearby Landmarks
```http
POST /api/get-landmarks
Content-Type: application/json

{
    "address": "123 Main St, Kanpur, 208001",
    "radius": 500
}
```

### 3. Health Check
```http
GET /health
```

## Features in Detail

### PIN Code Validation
- Validates 6-digit Indian PIN codes
- Matches PIN code with provided city
- Suggests correct city name if mismatched

### Landmark Discovery
- Finds up to 5 most relevant nearby landmarks
- Smart scoring based on:
  - Google Maps rating
  - Number of reviews
  - Distance from location
- Customizable search radius

### Frontend Interface
- Responsive design
- Step-by-step form validation
- Interactive landmark display
- Direct Google Maps integration

## Algorithm

### Landmark Scoring Formula
The popularity score for each landmark is calculated using:

```
PopularityScore = (Rating × log10(Reviews + 1)) / (1 + Distance/1000)
```

### Distance Calculation
Uses the Haversine formula for accurate distance calculation between two geographical points:

```
d = 2R × arcsin(sqrt(sin²(Δφ/2) + cos(φ1)cos(φ2)sin²(Δλ/2)))
```

Where:
- R is Earth's radius (6371 km)
- φ is latitude
- λ is longitude

## Project Structure

```
meesho_dice_challenge/
├── main.go          # Server and API implementation
├── static/          # Static frontend files
│   └── index.html   # Web interface
├── .env             # Environment configuration
└── README.md        # Project documentation
```