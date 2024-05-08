package main

import (
	"encoding/json" // JSON verilerini encode ve decode etmek için
	"fmt"
	"html/template" // HTML şablonları işlemek için
	"log"
	"net/http" // HTTP sunucuları ve istemciler oluşturmak için
	"strconv"
	"strings"
	"sync" // Senkronizasyon işlemleri için
	"time" // Zamanla ilgili işlemler için
)

type Artist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Image        string   `json:"image"`
	Year         int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Members      []string `json:"members"`
	LocationsURL string   `json:"locations"`
	Locations    []string
	ConcertDates string
	Dates        []time.Time
	RelationsUrl string `json:"relations"`
	Relations    []string
}

func main() {
	http.HandleFunc("/", handler)                                                                       // Ana sayfa için belirli bir işlev
	http.HandleFunc("/badrequest", handleBadRequest)                                                    // Diğer yollara gelen istekler için genel bad request işlevi
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("templates")))) // Şablonları işlemek için
	log.Println("Web sunucusu başlatıldı. http://localhost:8080 adresine gidiniz.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		handleBadRequest(w, r)
		return
	}

	apiUrl := "https://groupietrackers.herokuapp.com/api/artists"
	artists, err := getArtistsData(apiUrl)
	if err != nil {
		http.Error(w, "Sanatçı verisi alınamadı", http.StatusInternalServerError)
		return
	}

	artists, err = updateArtistInfo(artists)
	if err != nil {
		http.Error(w, "Sanatçı bilgileri güncellenemedi", http.StatusInternalServerError)
		return
	}

	searchQuery := r.URL.Query().Get("search")
	if searchQuery != "" {
		artists = filterArtists(artists, searchQuery)
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate != "" && endDate != "" {
		artists = filterArtistsByCreationDate(artists, startDate, endDate)
	}

	minFirstAlbumDate := r.URL.Query().Get("min_first_album_date")
	maxFirstAlbumDate := r.URL.Query().Get("max_first_album_date")

	if minFirstAlbumDate != "" && maxFirstAlbumDate != "" {
		artists = filterArtistsByFirstAlbumDateRange(artists, minFirstAlbumDate, maxFirstAlbumDate)
	}
	members := r.URL.Query()["members"]
	if len(members) > 0 {
		artists = filterArtistByMembersNumbers(artists, members)
	}

	renderTemplate(w, artists)
}

func filterArtistByMembersNumbers(artists []Artist, members []string) []Artist {
	var filteredArtists []Artist
	for _, artist := range artists {
		if containsMemberCount(artist, members) {
			filteredArtists = append(filteredArtists, artist)
		}
	}
	return filteredArtists
}

func containsMemberCount(artist Artist, members []string) bool {
	for _, count := range members {
		memberCount, err := strconv.Atoi(count)
		if err != nil {
			log.Println("Üye sayısı dönüştürülemedi:", err)
			continue
		}
		if len(artist.Members) == memberCount {
			return true
		}
	}
	return false
}

func filterArtistsByFirstAlbumDateRange(artists []Artist, mindate, maxdate string) []Artist {
	var filterArtists []Artist
	for _, artist := range artists {
		if filterArtistsByFirstAlbumDate(artist.FirstAlbum, mindate, maxdate) {
			filterArtists = append(filterArtists, artist)
		}
	}
	return filterArtists
}

func filterArtistsByFirstAlbumDate(dateStr, minDateStr, maxDateStr string) bool {
	date, err := time.Parse("02-01-2006", dateStr)
	if err != nil {
		log.Println("İlk albüm tarihi ayrıştırılamadı:", err)
		return false
	}

	minDate, err := time.Parse("2006-01-02", minDateStr) // Minimum tarih değerini uygun formata dönüştür
	if err != nil {
		log.Println("Min tarih ayrıştırılamadı:", err)
		return false
	}

	maxDate, err := time.Parse("2006-01-02", maxDateStr)
	if err != nil {
		log.Println("Max tarih ayrıştırılamadı:", err)
		return false
	}

	return date.After(minDate) && date.Before(maxDate)
}

func filterArtistsByCreationDate(artists []Artist, startDate, endDate string) []Artist {
	var filteredArtists []Artist
	startYear, err := strconv.Atoi(startDate)
	if err != nil {
		log.Println("Başlangıç tarihi hatalı:", err)
		return artists
	}
	endYear, err := strconv.Atoi(endDate)
	if err != nil {
		log.Println("Bitiş tarihi hatalı:", err)
		return artists
	}

	for _, artist := range artists {
		if artist.Year >= startYear && artist.Year <= endYear {
			filteredArtists = append(filteredArtists, artist)
		}
	}
	return filteredArtists
}

func handleBadRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Bad Request: İstek işlenemedi", http.StatusBadRequest)
}

func getArtistsData(apiUrl string) ([]Artist, error) {
	resp, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // hata oluşmazsa isteğin yanıtı alınır ve yanıt gövdesi kapatılır.

	var artists []Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, err
	}

	return artists, nil
}

func updateArtistInfo(artists []Artist) ([]Artist, error) {
	var wg sync.WaitGroup // Wait, tüm işlemlerin tamamlanmasını beklemek için kullanılır.
	var mu sync.Mutex     // Mutex, güncellemelerin eşzamanlı erişimini kontrol etmek için kullanılır

	for i, artist := range artists {
		wg.Add(1) // Her bir artist için bir goroutine başlatıldığında WaitGroup'e 1 ekleriz  --Gorountine: Go programlarının eşzamanlılığı sağlamak için kullanılır--
		go func(i int, artist Artist) {
			defer wg.Done() // Goroutine tamamlandığında WaitGroup'ten çıkar

			if len(artist.Locations) == 0 && artist.LocationsURL != "" {
				locations, err := fetchData(artist.LocationsURL)
				if err != nil {
					log.Printf("Konum bilgileri alınamadı: %s\n", err)
				} else {
					mu.Lock()
					artists[i].Locations = locations
					mu.Unlock()
				}
			}

			if len(artist.Dates) == 0 && artist.ConcertDates != "" {
				dates, err := fetchDates(artist.ConcertDates)
				if err != nil {
					log.Printf("Konser tarihleri alınamadı: %s\n", err)
				} else {
					mu.Lock()
					artists[i].Dates = dates
					mu.Unlock()
				}
			}

			if len(artist.Relations) == 0 && artist.RelationsUrl != "" {
				relations, err := fetchRelations(artist.RelationsUrl)
				if err != nil {
					log.Printf("İlişkiler alınamadı: %s\n", err)
				} else {
					mu.Lock()
					artists[i].Relations = relations
					mu.Unlock()
				}
			}
		}(i, artist)
	}

	wg.Wait() // Tüm gorutinlerin tamamlanmasını bekler

	return artists, nil
}

func fetchData(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Locations []string `json:"locations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	// Ayıklanan konum verilerini uygun bir biçime dönüştürme
	formattedLocations := make([]string, len(data.Locations))
	for i, location := range data.Locations {
		formattedLocation := formatLocationName(location)
		formattedLocations[i] = formattedLocation
	}

	return formattedLocations, nil
}

func formatLocationName(location string) string {
	parts := strings.Split(location, "-") // "-" karakterine göre ayır
	formattedParts := make([]string, len(parts))
	for i, part := range parts {
		formattedPart := strings.Title(strings.Replace(part, "_", " ", -1)) // Kelimenin ilk harfini büyük yap ve "_" karakterini boşlukla değiştir
		formattedParts[i] = formattedPart
	}
	return strings.Join(formattedParts, ", ") // Ayıklanan parçaları birleştir ve virgülle ayır
}

func fetchDates(url string) ([]time.Time, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Dates []string `json:"dates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var dates []time.Time
	for _, dateStr := range removeStarsFromDates(data.Dates) {
		date, err := time.Parse("02-01-2006", dateStr)
		if err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}

	return dates, nil
}

func fetchRelations(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		DatesLocations map[string][]string `json:"datesLocations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var relations []string
	for location, dates := range data.DatesLocations {
		for _, date := range dates {
			relation := fmt.Sprintf("%s: %s", location, date)
			relations = append(relations, relation)
		}
	}

	return relations, nil
}

func filterArtists(artists []Artist, searchQuery string) []Artist {
	var filteredArtists []Artist
	for _, artist := range artists {
		if containsSearchQuery(artist, searchQuery) {
			filteredArtists = append(filteredArtists, artist)
		}
	}
	return filteredArtists
}

func containsSearchQuery(artist Artist, searchQuery string) bool {
	for _, member := range artist.Members {
		if strings.Contains(strings.ToLower(member), strings.ToLower(searchQuery)) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(artist.Name), strings.ToLower(searchQuery)) {
		return true
	}
	if strconv.Itoa(artist.Year) == searchQuery {
		return true
	}
	if artist.FirstAlbum == searchQuery {
		return true
	}
	return false
}

func renderTemplate(w http.ResponseWriter, artists []Artist) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "HTML şablonu parse edilemedi", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, artists)
	if err != nil {
		http.Error(w, "HTML şablonu işlenemedi", http.StatusInternalServerError)
		return
	}
}

func removeStarsFromDates(dates []string) []string {
	var cleanedDates []string
	for _, date := range dates {
		cleanedDate := strings.TrimLeft(date, "*")
		cleanedDates = append(cleanedDates, cleanedDate)
	}
	return cleanedDates
}
