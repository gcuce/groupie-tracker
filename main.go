package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

type Artist struct {
	Name       string   `json:"name"`
	Image      string   `json:"image"`
	Year       int      `json:"year"`
	FirstAlbum string   `json:"firstAlbum"`
	Members    []string `json:"members"`
}

func main() {
	// Web sunucusunu başlat
	http.HandleFunc("/", handler)
	log.Println("Web sunucusu başlatıldı. http://localhost:8080 adresine gidin.")
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

	// HTML şablon dosyasını parse et
	t, err := template.ParseFiles("artists.html")
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
