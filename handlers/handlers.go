package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jung-kurt/gofpdf"
)

var history sync.Map
var count int64
var WG *sync.WaitGroup
var operations sync.Map

const datafile = "data.json"

type State struct {
	Count   int64              `json:"count"`
	History map[int64]Response `json:"history"`
}

type ListNum struct {
	Nums []int64 `json:"nums"`
}

type Request struct {
	Links []string `json:"links"`
}

type Response struct {
	Links    map[string]string `json:"links"`
	LinksNum int64             `json:"links_num"`
}

func init() {
	loadState()
}

func loadState() {
	data, err := os.ReadFile(datafile)
	if err != nil {
		log.Println("Файл состояния не найден, начинаем с пустого")
		return
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("Ошибка загрузки состояния: %v\n", err)
		return
	}

	count = state.Count
	for k, v := range state.History {
		history.Store(k, v)
	}

	log.Printf("Состояние загружено: %d записей, счетчик: %d\n", len(state.History), state.Count)
}

func SaveState() {
	state := State{
		Count:   count,
		History: make(map[int64]Response),
	}

	history.Range(func(key, value interface{}) bool {
		state.History[key.(int64)] = value.(Response)
		return true
	})

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Ошибка маршалинга состояния: %v\n", err)
		return
	}

	if err := os.WriteFile(datafile, data, 0644); err != nil {
		log.Printf("Ошибка сохранения состояния: %v\n", err)
		return
	}

	log.Printf("Состояние сохранено: %d записей\n", len(state.History))
}

func SetWaitGroup(wg *sync.WaitGroup) {
	WG = wg
}

func CheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Неверный JSON"}`, http.StatusBadRequest)
		return
	}

	if len(req.Links) == 0 {
		http.Error(w, `{"error": "Список ссылок пуст"}`, http.StatusBadRequest)
		return
	}

	linkNum := atomic.AddInt64(&count, 1)

	res := Response{
		Links:    make(map[string]string),
		LinksNum: linkNum,
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex

	operationID := fmt.Sprintf("check_%d", linkNum)
	if WG != nil {
		WG.Add(1)
		operations.Store(operationID, true)
	}

	for _, link := range req.Links {
		wg.Add(1)

		go func(link string) {
			defer wg.Done()

			client := &http.Client{
				Timeout: 20 * time.Second,
			}

			normLink := normalizeURL(link)
			resp, err := client.Get(normLink)

			mutex.Lock()
			defer mutex.Unlock()

			if err != nil {
				res.Links[link] = "not available"
			} else {
				defer resp.Body.Close()
				if resp.StatusCode < 400 {
					res.Links[link] = "available"
				} else {
					res.Links[link] = "not available"
				}
			}
		}(link)
	}

	wg.Wait()

	if WG != nil {
		operations.Delete(operationID)
		WG.Done()
	}

	history.Store(linkNum, res)
	SaveState()
	json.NewEncoder(w).Encode(res)
	fmt.Println(res)
}

func ListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	var nums ListNum
	if err := json.NewDecoder(r.Body).Decode(&nums); err != nil {
		http.Error(w, `{"error": "Неверный JSON"}`, http.StatusBadRequest)
		return
	}

	if len(nums.Nums) == 0 {
		http.Error(w, `{"error": "Список номеров пуст"}`, http.StatusBadRequest)
		return
	}

	minNum := nums.Nums[0]
	maxNum := nums.Nums[len(nums.Nums)-1]

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Ln(12)

	for linkNum := minNum; linkNum <= maxNum; linkNum++ {
		if res, exists := history.Load(linkNum); exists {
			resp := res.(Response)

			pdf.SetFont("Arial", "B", 14)
			pdf.Cell(40, 10, fmt.Sprintf("Set #%d", linkNum))
			pdf.Ln(8)

			pdf.SetFont("Arial", "", 12)

			for link, status := range resp.Links {
				pdf.Cell(40, 10, fmt.Sprintf("%s: %s", link, status))
				pdf.Ln(6)
			}
			pdf.Ln(10)
		} else {
			pdf.SetFont("Arial", "", 12)
			pdf.Cell(40, 10, fmt.Sprintf("Набор #%d не найден", linkNum))
			pdf.Ln(10)
		}
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=links_report.pdf")

	err := pdf.Output(w)
	if err != nil {
		http.Error(w, `{"error": "Ошибка генерации PDF"}`, http.StatusInternalServerError)
		return
	}
}

func normalizeURL(link string) string {
	link = strings.TrimSpace(link)

	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		return "https://" + link
	}
	return link
}
