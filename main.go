package main

import (
	"encoding/csv"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type CanvasRequest struct {
	url         string
	method      string
	headers     map[string]string
	dataType    string
	data        map[string]string
	sis_user_id string
}

type ResponseResult struct {
	msg  string
	code int
}

func readCSV(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening CSV file:", err)
		return nil, err
	}

	reader := csv.NewReader(file)
	headerRow, err := reader.Read()
	if err != nil {
		return nil, err
	} else {
		log.Default().Println(headerRow)
	}

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return nil, err
	}
	closeErr := file.Close()
	if closeErr != nil {
		fmt.Println("Error reading CSV:", err)
		return nil, err
	}
	attendData := make(map[string]string)

	for _, record := range records {
		attendData[record[0]] = record[1]
	}

	return attendData, nil
}

func buildRequests(data map[string]string, courseNumber int, assignmentNumber int) ([]CanvasRequest, error) {
	reqTotal := len(data)
	reqCnt := 0
	reqs := make([]CanvasRequest, reqTotal)
	baseDomain := os.Getenv("CANVAS_DOMAIN")
	apikey := os.Getenv("CANVAS_API")
	for studentId, magicWord := range data {
		reqs[reqCnt] = CanvasRequest{
			url:    fmt.Sprintf("%s/courses/%d/assignments/%d/submissions/sis_user_id:%s", baseDomain, courseNumber, assignmentNumber, studentId),
			method: "PUT",
			data: map[string]string{
				"comment[text_comment]":    magicWord,
				"submission[posted_grade]": "1",
			},
			dataType: "text",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", apikey),
				"Content-Type":  "application/x-www-form-urlencoded",
			},
			sis_user_id: studentId,
		}
		reqCnt++
	}
	return reqs, nil
}

func sendRequests(client *http.Client, reqs []CanvasRequest) ([]ResponseResult, error) {
	results := make([]ResponseResult, len(reqs))
	for i := range reqs {
		if i%15 == 0 {
			time.Sleep(time.Second)
		}
		var testReq = reqs[i]

		payload := url.Values{}

		for key, value := range testReq.data {
			payload.Add(key, value)
		}

		putReq, err := http.NewRequest(testReq.method, testReq.url, strings.NewReader(payload.Encode()))
		if err != nil {
			results[i] = ResponseResult{
				msg:  fmt.Sprintf("Failed to make new request\n\t%v\n", err),
				code: 500,
			}
		}
		for key, value := range testReq.headers {
			putReq.Header.Set(key, value)
		}
		resp, err := client.Do(putReq)
		if err != nil {
			results[i] = ResponseResult{
				msg:  fmt.Sprintf("Failed to start request\n\t%v\n", err),
				code: 500,
			}
		}

		if resp.Status != "200 OK" {

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results[i] = ResponseResult{
					msg:  fmt.Sprintf("Failed to read request body\n\t%v\n", body),
					code: 500,
				}
			}
			fmt.Printf("Upload for %s failed: %v\n", testReq.sis_user_id, string(body))
		}
		_ = resp.Body.Close()
	}

	return results, nil
}

func main() {
	logger := log.New(os.Stderr, "", 0)
	var rootCmd = &cobra.Command{
		Use:   "csubmit [flags] <CSVFile>",
		Short: "Upload canvas assignment for a course",
		Long:  "Upload canvas assignment for a course by specifying Course ID, Assignment ID,\nand the path to a CSV file containing attendance data.\nBy default, the student id is in col 0 and score is col 1, comments are in col 2",
	}

	var courseID int
	var assignID int
	var csvFile string
	var headerRow int
	var scoreColumn int

	rootCmd.Flags().IntVarP(&courseID, "courseid", "c", 0, "Course ID")
	rootCmd.Flags().IntVarP(&assignID, "assignid", "a", 0, "Attendance Assignment ID")
	rootCmd.Flags().IntVarP(&headerRow, "headerow", "r", 0, "Index of header row, default 0")
	rootCmd.Flags().IntVarP(&scoreColumn, "scorecol", "s", 1, "Index of score column")

	rootCmd.Run = func(cmd *cobra.Command, args []string) {

		if courseID <= 0 || assignID <= 0 || csvFile == "" {
			err := cmd.Help()
			if err != nil {
				logger.Printf("Error Running command %v", err)
				os.Exit(-2)
			}
			return
		}

		fmt.Printf("Recording attendance for Course ID: %d, Attendance ID: %d, CSV File: %s\n", courseID, assignID, csvFile)
		// Implement your logic to record attendance here.
		data, err := readCSV(csvFile)
		if err != nil {
			logger.Printf("Error Running command %v", err)
			os.Exit(-2)
		}
		builtReqs, err := buildRequests(data, courseID, assignID)
		if err != nil {
			logger.Printf("Error Running command %v", err)
			os.Exit(-2)
		}
		//start to send reqs
		client := &http.Client{}
		cnt, err := sendRequests(client, builtReqs)
		if err != nil {
			logger.Printf("Error Running command %v", err)
			os.Exit(-2)
		}
		fmt.Printf("%d requests were sent\n", len(cnt))
	}

	rootCmd.Args = cobra.ExactArgs(1)
	rootCmd.Args = func(cmd *cobra.Command, args []string) error {

		csvFile = args[0]
		return nil
	}

	if err := rootCmd.Execute(); err != nil {
		logger.Printf("Error Running command %v", err)
		os.Exit(-2)
	}
}
