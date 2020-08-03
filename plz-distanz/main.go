package main

import "encoding/csv"
import "log"
import "os"
import "sort"
import "strconv"
//import "github.com/davecgh/go-spew/spew"

type Entry struct {
	Longitude	float64
	Latitude	float64
	Name		string
}

func main() {
	db, err := os.Open("./PLZ.tab")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	reader := csv.NewReader(db)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1

	// Read header and discard
	_, _  = reader.Read()

	// Read actual data
	dbdata, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	AllEntries := make(map[int]*Entry)

	for _, line := range dbdata {
		plz, _ := strconv.Atoi(line[1])
		AllEntries[plz] = &Entry{}
		AllEntries[plz].Longitude, _ = strconv.ParseFloat(line[2], 32)
		AllEntries[plz].Latitude, _ = strconv.ParseFloat(line[3], 32)
		AllEntries[plz].Name = line[4]
	}

	// Regierungssitz Berlin -> Bonn
	log.Println(distance(AllEntries[10557].Latitude, AllEntries[10557].Longitude, AllEntries[53113].Latitude, AllEntries[53113].Longitude, `K`))
	// Should return 63303, 81249, 22527
	log.Println(find_closest(AllEntries, 60313, []int{22527,81249,63303}, 3))
}

func find_closest (AllEntries map[int]*Entry, destination int, candidates []int, results int) ([]int) {
	// Cap the number of results to the number of inputs
	if results > len(candidates) { results = len(candidates) }

	var result []int
	distances := make(map[float64]int)

	for _, candidate := range candidates {
		kilometers := distance(AllEntries[destination].Latitude,
		                       AllEntries[destination].Longitude,
                                       AllEntries[candidate].Latitude,
				       AllEntries[candidate].Longitude,
				       `K`)
                distances[kilometers] = candidate
	}

	keys := make([]float64, 0, len(distances))
	for k := range distances {
		keys = append(keys, k)
	}
	sort.Float64s(keys)

	for _, r := range keys {
		result = append(result, distances[r])
	}

	return result[0:results]
}
