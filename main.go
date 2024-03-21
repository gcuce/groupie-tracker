package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Artist struct {
	Name         string   `json:"name"`
	Image        string   `json:"image"`
	Year         int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Members      []string `json:"members"`
	Locations    string   `json:"locations"`
	ConcertDates string   `json:"concertDates"`
	Relations    string   `json:"relations"`
}

func main() {
	// Statik dosyaları sunucuya tanımla
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("templates"))))

	// HTTP Sunucusu başlat
	log.Println("Web sunucusu başlatıldı. http://localhost:8080 adresine gidin.")
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	// API endpoint URL
	apiUrl := "https://groupietrackers.herokuapp.com/api/artists"

	// HTTP isteği yap
	resp, err := http.Get(apiUrl)
	if err != nil {
		http.Error(w, "API isteği başarısız", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// İstekten gelen veriyi oku
	var artists []Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		http.Error(w, "JSON verisi okunamadı", http.StatusInternalServerError)
		return
	}
	// Arama terimini al
	searchQuery := r.URL.Query().Get("search")
	if searchQuery != "" {
		filteredArtists := []Artist{}
		for _, artist := range artists {
			memberMatch := false
			for _, member := range artist.Members {
				if strings.Contains(strings.ToLower(member), strings.ToLower(searchQuery)) {
					memberMatch = true
					break
				}
			}
			if memberMatch || strings.Contains(strings.ToLower(artist.Name), strings.ToLower(searchQuery)) {
				filteredArtists = append(filteredArtists, artist)
			} else if strconv.Itoa(artist.Year) == searchQuery {
				filteredArtists = append(filteredArtists, artist)
			} else if artist.FirstAlbum == searchQuery {
				filteredArtists = append(filteredArtists, artist)
			}
		}

		artists = filteredArtists
	}

	// HTML şablon dosyasını parse et
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "HTML şablonu parse edilemedi", http.StatusInternalServerError)
		return
	}

	// Sanatçı bilgilerini HTML şablonuna uygula
	if err := t.Execute(w, artists); err != nil {
		http.Error(w, "HTML şablonu işlenemedi", http.StatusInternalServerError)
		return
	}

}
