package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const serverURL = "http://srv.msk01.gigacorp.local/_stats"

const (
	LoadAverageThreshold   = 30.0
	RAMUsedThreshold       = 80.0
	DiskUsedThreshold      = 90.0
	NetworkBwUsedThreshold = 90.0

	BytesInMegabyte = 1024 * 1024
	BitsInMegabit   = 1000 * 1000
)

type ServerStats struct {
	LoadAverage    float64
	RAMTotal       int64
	RAMUsed        int64
	DiskTotal      int64
	DiskUsed       int64
	NetworkBwTotal int64
	NetworkBwUsed  int64
}

func parseStats(data []byte) (*ServerStats, error) {
	s := strings.TrimSpace(string(data))

	parts := strings.Split(s, ",")

	if len(parts) != 7 {
		return nil, fmt.Errorf("ожидалось 7 полей, получено %d", len(parts))
	}

	stats := &ServerStats{}
	var err error

	stats.LoadAverage, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить LoadAverage: %w", err)
	}

	stats.RAMTotal, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить RAMTotal: %w", err)
	}

	stats.RAMUsed, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить RAMUsed: %w", err)
	}

	stats.DiskTotal, err = strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить DiskTotal: %w", err)
	}

	stats.DiskUsed, err = strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить DiskUsed: %w", err)
	}

	stats.NetworkBwTotal, err = strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить NetworkBwTotal: %w", err)
	}

	stats.NetworkBwUsed, err = strconv.ParseInt(parts[6], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить NetworkBwUsed: %w", err)
	}

	return stats, nil
}

func checkStats(stats *ServerStats) {
	if stats.LoadAverage > LoadAverageThreshold {
		fmt.Printf("Load Average is too high: %.0f\n", stats.LoadAverage)
	}
	if stats.RAMTotal > 0 {
		ramUsedPercent := (float64(stats.RAMUsed) / float64(stats.RAMTotal)) * 100
		if RAMUsedThreshold < ramUsedPercent {
			fmt.Printf("Memory usage too high: %.0f%%\n", ramUsedPercent)
		}
	}
	if stats.DiskTotal > 0 {
		dickUsedPercent := (float64(stats.DiskUsed) / float64(stats.DiskTotal)) * 100
		if DiskUsedThreshold < dickUsedPercent {
			freeSpaceBytes := stats.DiskTotal - stats.DiskUsed
			freeSpaceMb := freeSpaceBytes / BytesInMegabyte
			fmt.Printf("Free disk space is too low: %d Mb left\n", freeSpaceMb)
		}
	}
	if stats.NetworkBwTotal > 0 {
		netUsedPercent := (float64(stats.NetworkBwUsed) / float64(stats.NetworkBwTotal)) * 100
		if NetworkBwUsedThreshold < netUsedPercent {
			availableBps := stats.NetworkBwTotal - stats.NetworkBwUsed
			availableMbps := float64(availableBps) / float64(BitsInMegabit)
			fmt.Printf("Network bandwidth usage high: %.0f Mbit/s available\n", math.RoundToEven(availableMbps))
		}
	}
}

func performCheck() error {
	resp, err := http.Get(serverURL)
	if err != nil {
		return fmt.Errorf("ошибка ри выполнении запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер вернул некорректный статус: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка при парсинге ответа: %w", err)
	}

	stats, err := parseStats(body)
	if err != nil {
		return fmt.Errorf("ошибка при парсинге данных сервера: %w", err)
	}

	checkStats(stats)
	return nil
}

func main() {
	const checkInterval = 1 * time.Second
	const errorThreshold = 3

	var errorsCount int

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	if err := performCheck(); err != nil {
		log.Printf("Chack failed: %v", err)
		errorsCount++
	}
	for range ticker.C {
		if err := performCheck(); err != nil {
			log.Printf("Chack failed: %v", err)
			errorsCount++
			if errorsCount >= errorThreshold {
				fmt.Println("Unable to fetch server statistic")
			}
		} else {
			if errorsCount > 0 {
				errorsCount = 0
			}
		}
	}

}
