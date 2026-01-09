package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o parse-rtc-charge-log

// 对应 JSON 结构的 Go struct（仅定义你需要的字段）
type LogRecord struct {
	Name   string `json:"name"`
	Fields struct {
		Bytes    int64 `json:"bytes"`
		Duration int64 `json:"duration"`
		Height   int64 `json:"height,omitempty"`
		Width    int64 `json:"width,omitempty"`
	} `json:"fields"`
	Tags struct {
		AppId string `json:"appId"`
		// Group         string `json:"group"`
		Method string `json:"method"`
		// PlayerId      string `json:"playerId"`
		// RoomIId       string `json:"roomIId"`
		RoomId string `json:"roomId"`
		// RoomServerIId string `json:"roomServerIId"`
		// RoomServerId  string `json:"roomServerId"`
		// StateCenterId string `json:"stateCenterId"`
		Time    string `json:"time"` // 注意：这是字符串时间戳
		TrackId string `json:"trackId"`
		Type    string `json:"type"`
		Uid     string `json:"uid"`
		Uuid    string `json:"uuid"`
	} `json:"tags"`
	Ts int64 `json:"ts"` // nanosecond timestamp
}

type DateDurationsType map[string]int64
type DateRoomsType map[string]map[string]int64

// 存储每个日期的duration总和
var mapDateDurationsAudio = make(DateDurationsType)
var mapDataDurationsVidSd = make(DateDurationsType)
var mapDataDurationsVidHd = make(DateDurationsType)
var mapDataDurationsVidUhd = make(DateDurationsType)

// 存储每个日期的所有房间ID
var mapDataRoomsAudio = make(DateRoomsType)
var mapDataRoomsVidSd = make(DateRoomsType)
var mapDataRoomsVidHd = make(DateRoomsType)
var mapDataRoomsVidUhd = make(DateRoomsType)

var year = 2025
var month = 12
var day = 27
var date = "2025-12-27"
var appid = "icha9jt73"
var roomId = ""

func main() {
	fmt.Printf("./parse-rtc-charge-log 2025 12 1 30 g1xr9210d [roomId]\n")
	fmt.Printf("./parse-rtc-charge-log 2025 12 1 30 g1xr9210d [roomId]\n")
	fmt.Printf("./parse-rtc-charge-log 2025 12 1 30 g1xr9210d [roomId]\n")
	fmt.Printf("./parse-rtc-charge-log 2025 12 1 30 g1xr9210d [roomId]\n")

	var err error

	yearStr := os.Args[1]
	year, err = strconv.Atoi(yearStr)
	if err != nil {
		fmt.Printf("invalid year: %s\n", yearStr)
		os.Exit(1)
	}

	monthStr := os.Args[2]
	month, err = strconv.Atoi(monthStr)
	if err != nil {
		fmt.Printf("invalid month: %s\n", monthStr)
		os.Exit(1)
	}
	if month < 1 || month > 12 {
		fmt.Printf("invalid month: %d\n", month)
		os.Exit(1)
	}

	minDayStr := os.Args[3]
	minDay, err := strconv.Atoi(minDayStr)
	if err != nil {
		fmt.Printf("invalid day: %s\n", minDayStr)
		os.Exit(1)
	}
	if minDay < 1 || minDay > 31 {
		fmt.Printf("invalid day: %d\n", minDay)
		os.Exit(1)
	}
	maxDayStr := os.Args[4]
	maxDay, err := strconv.Atoi(maxDayStr)
	if err != nil {
		fmt.Printf("invalid max day: %s\n", maxDayStr)
		os.Exit(1)
	}
	if maxDay < 1 || maxDay > 31 {
		fmt.Printf("invalid max day: %d\n", maxDay)
		os.Exit(1)
	}

	appid = os.Args[5]
	if len(os.Args) > 6 {
		roomId = os.Args[6]
	}

	// rootDir := "/pili-logs/2025-12-27/json_rtc_charge"
	// rootDir := "./"
	// procDir(rootDir)

	for day = minDay; day <= maxDay; day++ {
		date = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		rootDir := fmt.Sprintf("/pili-logs/%s/json_rtc_charge", date)
		procDir(rootDir)
	}

	showResult()
}

func getVideoStd(w, h int64) string {
	sizeMul := w * h
	if sizeMul <= 640*360 {
		return "sd"
	}
	if sizeMul <= 1280*720 {
		return "hd"
	}
	return "uhd"
}

func getDateDurationsType(mt string, w, h int64) *DateDurationsType {
	if mt == "audio" {
		return &mapDateDurationsAudio
	}

	if mt == "video" {
		vs := getVideoStd(w, h)
		if vs == "sd" {
			return &mapDataDurationsVidSd
		}
		if vs == "hd" {
			return &mapDataDurationsVidHd
		}
		if vs == "uhd" {
			return &mapDataDurationsVidUhd
		}
	}
	return nil
}

func getDateRoomsType(mt string, w, h int64) *DateRoomsType {
	if mt == "audio" {
		return &mapDataRoomsAudio
	}
	if mt == "video" {
		vs := getVideoStd(w, h)
		if vs == "sd" {
			return &mapDataRoomsVidSd
		}
		if vs == "hd" {
			return &mapDataRoomsVidHd
		}
		if vs == "uhd" {
			return &mapDataRoomsVidUhd
		}
	}
	return nil
}

func procDir(rootDir string) {

	// 获取rootDir的内容
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading root directory: %v\n", err)
		return
	}

	// 遍历每个条目
	for _, entry := range entries {
		entryPath := filepath.Join(rootDir, entry.Name())

		// 只处理直接子目录
		if entry.IsDir() {
			// 读取子目录中的内容
			subEntries, err := os.ReadDir(entryPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading subdirectory %s: %v\n", entryPath, err)
				continue
			}

			// 遍历子目录中的每个条目
			for _, subEntry := range subEntries {
				subEntryPath := filepath.Join(entryPath, subEntry.Name())

				// 只处理子目录中的.log文件
				if !subEntry.IsDir() && strings.HasSuffix(subEntry.Name(), ".log") {
					fmt.Fprintf(os.Stderr, "Processing: %s\n", subEntryPath)
					err := processLogFile(subEntryPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", subEntryPath, err)
					}
				}
			}
		}
	}
}

func showResult() {
	fmt.Printf("\n########## appid: %s, type: audio, durations\n", appid)
	totalDuration := int64(0)
	for date, duration := range mapDateDurationsAudio {
		fmt.Printf("%s: %ds, %dm, %.2fh\n", date, duration, duration/60, float64(duration)/3600)
		totalDuration += duration
	}
	fmt.Printf("########## total durations: %ds, %dm, %.2fh\n", totalDuration, totalDuration/60, float64(totalDuration)/3600)

	fmt.Printf("\n########## appid: %s, type: video, size: SD, durations\n", appid)
	totalDuration = 0
	for date, duration := range mapDataDurationsVidSd {
		fmt.Printf("%s: %ds, %dm, %.2fh\n", date, duration, duration/60, float64(duration)/3600)
		totalDuration += duration
	}
	fmt.Printf("########## total durations: %ds, %dm, %.2fh\n", totalDuration, totalDuration/60, float64(totalDuration)/3600)

	fmt.Printf("\n########## appid: %s, type: video, size: HD, durations\n", appid)
	for date, duration := range mapDataDurationsVidHd {
		fmt.Printf("%s: %ds, %dm, %.2fh\n", date, duration, duration/60, float64(duration)/3600)
		totalDuration += duration
	}
	fmt.Printf("########## total durations: %ds, %dm, %.2fh\n", totalDuration, totalDuration/60, float64(totalDuration)/3600)

	fmt.Printf("\n########## appid: %s, type: video, size: UHD, durations\n", appid)
	for date, duration := range mapDataDurationsVidUhd {
		fmt.Printf("%s: %ds, %dm, %.2fh\n", date, duration, duration/60, float64(duration)/3600)
		totalDuration += duration
	}
	fmt.Printf("########## total durations: %ds, %dm, %.2fh\n", totalDuration, totalDuration/60, float64(totalDuration)/3600)

	type kv struct {
		Key   string
		Value int64
	}

	sortRoom := func(rooms map[string]int64) []kv {
		var sorted []kv
		for k, v := range rooms {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Value > sorted[j].Value // 注意：> 表示降序
		})
		return sorted
	}

	fmt.Printf("\n########## appid: %s, type: audio, rooms\n", appid)
	for date, rooms := range mapDataRoomsAudio {
		fmt.Printf("\n########## data: %s, appid: %s, type: audio, rooms: %d\n", date, appid, len(rooms))
		sorted := sortRoom(rooms)
		for _, item := range sorted {
			fmt.Printf("%s: room: %s, %.2fh\n", date, item.Key, float64(item.Value)/3600)
		}
	}

	fmt.Printf("\n########## appid: %s, type: video, size: SD, rooms\n", appid)
	for date, rooms := range mapDataRoomsVidSd {
		fmt.Printf("\n########## data: %s, appid: %s, type: video, size: SD, rooms: %d\n", date, appid, len(rooms))
		sorted := sortRoom(rooms)
		for _, item := range sorted {
			fmt.Printf("%s: room: %s, %.2fh\n", date, item.Key, float64(item.Value)/3600)
		}
	}

	fmt.Printf("\n########## appid: %s, type: video, size: HD, rooms\n", appid)
	for date, rooms := range mapDataRoomsVidHd {
		fmt.Printf("\n########## data: %s, appid: %s, type: video, size: HD, rooms: %d\n", date, appid, len(rooms))
		sorted := sortRoom(rooms)
		for _, item := range sorted {
			fmt.Printf("%s: room: %s, %.2fh\n", date, item.Key, float64(item.Value)/3600)
		}
	}

	fmt.Printf("\n########## appid: %s, type: video, size: UHD, rooms\n", appid)
	for date, rooms := range mapDataRoomsVidUhd {
		fmt.Printf("\n########## data: %s, appid: %s, type: video, size: UHD, rooms: %d\n", date, appid, len(rooms))
		sorted := sortRoom(rooms)
		for _, item := range sorted {
			fmt.Printf("%s: room: %s, %.2fh\n", date, item.Key, float64(item.Value)/3600)
		}
	}
}

func getRoomId(s string) string {
	var roomId string
	if parts := strings.Split(s, ":"); len(parts) >= 2 {
		roomId = parts[1]
	} else {
		// 如果没有冒号，使用原始值
		roomId = s
	}
	return roomId
}

func procChargeLog(record *LogRecord) {
	if record.Tags.Method != "play" {
		return
	}
	if record.Tags.AppId != appid {
		return
	}

	mtype := record.Tags.Type

	var dateDurations *DateDurationsType
	var dateRooms *DateRoomsType
	if mtype == "audio" {
		dateDurations = getDateDurationsType(mtype, 0, 0)
		dateRooms = getDateRoomsType(mtype, 0, 0)
		if dateDurations == nil || dateRooms == nil {
			return
		}
	} else if mtype == "video" {
		dateDurations = getDateDurationsType(mtype, record.Fields.Width, record.Fields.Height)
		dateRooms = getDateRoomsType(mtype, record.Fields.Width, record.Fields.Height)
		if dateDurations == nil || dateRooms == nil {
			return
		}
	} else {
		return
	}

	room := getRoomId(record.Tags.RoomId)
	if room == "" {
		return
	}
	showInfo := false
	if roomId != "" {
		if roomId != room {
			return
		}
		showInfo = true
	}

	sec, err := strconv.Atoi(record.Tags.Time)
	if err != nil {
		fmt.Printf("#### invalid record.Tags.Time: %s\n", record.Tags.Time)
		return
	}

	// dateStr := time.Unix(int64(sec), 0).UTC().Format("2006-01-02")
	dateStr := time.Unix(int64(sec), 0).Local().Format("2006-01-02")

	duration := record.Fields.Duration
	if v, ok := (*dateDurations)[dateStr]; ok {
		(*dateDurations)[dateStr] = v + duration
	} else {
		(*dateDurations)[dateStr] = duration
	}

	if v, ok := (*dateRooms)[dateStr]; ok {
		if durations, ok := v[room]; ok {
			(*dateRooms)[dateStr][room] = durations + duration
		} else {
			(*dateRooms)[dateStr][room] = duration
		}
	} else {
		(*dateRooms)[dateStr] = make(map[string]int64)
		(*dateRooms)[dateStr][room] = duration
	}

	// 打印整个结构体（格式化输出）
	if showInfo {
		fmt.Printf("%+v\n", record)
	}
}

func processLogFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if appid != "" {
			if !strings.Contains(line, appid) {
				continue
			}
		}

		var record LogRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			// 跳过非法 JSON 行，可选：记录警告
			fmt.Fprintf(os.Stderr, "Invalid JSON at %s:%d - %v\n", filePath, lineNum, err)
			continue
		}

		procChargeLog(&record)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
