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

// 存储每个日期的duration总和
var mapDateDurations = make(map[string]int64)

// 存储每个日期的所有房间ID
var mapDataRooms = make(map[string]map[string]int64)

var year = 2025
var month = 12
var day = 27
var date = "2025-12-27"
var appid = "icha9jt73"
var mtype = "video"
var videoSize = ""
var roomId = ""
var minWidth = int64(0)
var minHeight = int64(0)

func main() {
	fmt.Printf("./parse-rtc-charge-log 2025 12 1 30 g1xr9210d audio [roomId]\n")
	fmt.Printf("./parse-rtc-charge-log 2025 12 1 30 g1xr9210d video sd [roomId]\n")
	fmt.Printf("./parse-rtc-charge-log 2025 12 1 30 g1xr9210d video hd [roomId]\n")
	fmt.Printf("./parse-rtc-charge-log 2025 12 1 30 g1xr9210d video uhd [roomId]\n")

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
	mtype = os.Args[6]
	if mtype == "video" {
		videoSize = os.Args[7]
		if len(os.Args) > 8 {
			roomId = os.Args[8]
		}
	} else if mtype == "audio" {
		minWidth = int64(0)
		minHeight = int64(0)
		if len(os.Args) > 7 {
			roomId = os.Args[7]
		}
	} else {
		fmt.Printf("invalid mtype: %s\n", mtype)
		os.Exit(1)
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
	fmt.Printf("\n########## appid: %s, type:%s, size:%s, durations\n", appid, mtype, videoSize)
	for date, duration := range mapDateDurations {
		fmt.Printf("%s: %ds, %dm, %.2fh\n", date, duration, duration/60, float64(duration)/3600)
	}

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

	for date, rooms := range mapDataRooms {
		fmt.Printf("\n########## data: %s, appid: %s, type:%s, size:%s, rooms:%d\n", date, appid, mtype, videoSize, len(rooms))
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
	if record.Tags.Type != mtype {
		return
	}
	if mtype == "video" {
		sizeMul := record.Fields.Width * record.Fields.Height
		switch videoSize {
		case "sd":
			if sizeMul > 640*360 {
				return
			}
		case "hd":
			if sizeMul > 1280*720 || sizeMul <= 640*360 {
				return
			}
		case "uhd":
			if sizeMul <= 1280*720 {
				return
			}
		default:
			fmt.Printf("invalid video size: %s\n", videoSize)
			os.Exit(1)
		}

		if record.Fields.Width*record.Fields.Height <= minWidth*minHeight {
			return
		}
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
	if v, ok := mapDateDurations[dateStr]; ok {
		mapDateDurations[dateStr] = v + duration
	} else {
		mapDateDurations[dateStr] = duration
	}

	if v, ok := mapDataRooms[dateStr]; ok {
		if durations, ok := v[room]; ok {
			mapDataRooms[dateStr][room] = durations + duration
		} else {
			mapDataRooms[dateStr][room] = duration
		}
	} else {
		mapDataRooms[dateStr] = make(map[string]int64)
		mapDataRooms[dateStr][room] = duration
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
