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
	"sync"
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

type AssignmentFeedback struct {
	studentId string
	score     string
	comment   string
}

type ReaderOptions struct {
	headerRow    int
	scoreCol     int
	commentCol   int
	studentIdCol int
}

func readCSVDataFromSTDIN(stdin_stream io.Reader) (map[string]AssignmentFeedback, error) {
	return nil, nil
}
func readCSV(filename string, options ReaderOptions) (map[string]AssignmentFeedback, error) {

	file, err := os.Open(filename)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error opening CSV file: %v\n", err)
		return nil, err
	}

	reader := csv.NewReader(file)
	for i := 0; i < options.headerRow-1; i++ {
		_, err := reader.Read()
		if err != nil {
			return nil, err
		}
	}
	headerRow, err := reader.Read()
	if err != nil {
		return nil, err
	}
	sizeOfRow := len(headerRow)
	if sizeOfRow < options.scoreCol || sizeOfRow < options.studentIdCol || sizeOfRow < options.commentCol {
		return nil, fmt.Errorf("invalid index used for given reader options. Row length %d , Options: %v\n", sizeOfRow, options)
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
	attendData := make(map[string]AssignmentFeedback)

	for _, record := range records {
		attendData[record[options.studentIdCol]] = AssignmentFeedback{
			studentId: record[options.studentIdCol],
			score:     record[options.scoreCol],
			comment:   record[options.commentCol],
		}
	}

	return attendData, nil
}

func buildRequests(data map[string]AssignmentFeedback, courseNumber int, assignmentNumber int) ([]CanvasRequest, error) {
	reqTotal := len(data)
	reqCnt := 0
	reqs := make([]CanvasRequest, reqTotal)
	baseDomain := os.Getenv("CANVAS_DOMAIN")
	apikey := os.Getenv("CANVAS_TOKEN")
	if baseDomain == "" || apikey == "" {
		return nil, fmt.Errorf("canvas API and Domain not set in environment")
	}
	for studentId, feedbackInfo := range data {
		reqs[reqCnt] = CanvasRequest{
			url:    fmt.Sprintf("%s/courses/%d/assignments/%d/submissions/sis_user_id:%s", baseDomain, courseNumber, assignmentNumber, studentId),
			method: "PUT",
			data: map[string]string{
				"comment[text_comment]":    feedbackInfo.comment,
				"submission[posted_grade]": feedbackInfo.score,
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

func sendRequests(client *http.Client, reqs []CanvasRequest, outChan *chan ResponseResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for i := range reqs {
		var testReq = reqs[i]
		payload := url.Values{}

		for key, value := range testReq.data {
			payload.Add(key, value)
		}

		putReq, err := http.NewRequest(testReq.method, testReq.url, strings.NewReader(payload.Encode()))
		if err != nil {
			*outChan <- ResponseResult{
				code: 500,
				msg:  fmt.Sprintf("Failed to make new request\n\t%v\n", err),
			}
		}
		for key, value := range testReq.headers {
			putReq.Header.Set(key, value)
		}
		resp, err := client.Do(putReq)
		if err != nil {
			*outChan <- ResponseResult{
				code: 500,
				msg:  fmt.Sprintf("Failed to start request\n\t%v\n", err),
			}
		}

		if resp.Status != "200 OK" {

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				*outChan <- ResponseResult{
					code: 500,
					msg:  fmt.Sprintf("Failed to read request body\n\t%v\n", body),
				}
			} else {
				*outChan <- ResponseResult{
					code: resp.StatusCode,
					msg:  fmt.Sprintf("Upload for %s failed: %v\n", testReq.sis_user_id, string(body)),
				}
			}
		} else {
			*outChan <- ResponseResult{
				code: resp.StatusCode,
				msg:  fmt.Sprintf("Upload for %s success: %v\n", testReq.sis_user_id, resp.Status),
			}
		}
		_ = resp.Body.Close()
	}
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
	var studentIdCol int
	var scoreCol int
	var commentCol int

	rootCmd.Flags().IntVarP(&courseID, "courseid", "c", 0, "Course ID (Required)")
	rootCmd.Flags().IntVarP(&assignID, "assignid", "a", 0, "Attendance Assignment ID (Required)")
	rootCmd.Flags().IntVarP(&headerRow, "headerow", "r", 0, "Index of header row (default 0)")
	rootCmd.Flags().IntVarP(&studentIdCol, "sid", "i", 0, "Index of student id column (default 0)")
	rootCmd.Flags().IntVarP(&scoreCol, "score", "s", 1, "Index of score column")
	rootCmd.Flags().IntVarP(&commentCol, "comment", "t", 2, "Index of comment column")
	var err = rootCmd.MarkFlagRequired("courseid")
	if err != nil {
		return
	}
	err = rootCmd.MarkFlagRequired("assignid")
	if err != nil {
		return
	}

	rootCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			csvFile = args[0]
		} else if len(args) == 0 || (courseID <= 0 || assignID <= 0 || headerRow <= 0 || scoreCol <= 0 || commentCol <= 0 || csvFile == "") {
			fmt.Printf("Invalid usage!\n")
			err := cmd.Help()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error Running command %v", err)
				os.Exit(-2)
			}
			os.Exit(-1)
			return
		}
	}
	rootCmd.Run = func(cmd *cobra.Command, args []string) {

		fmt.Printf("Course Id: %d\nAssign Id: %d\nHeader Row index: %d\nStudent Id Col: %d\nScore Col: %d\nComment Col: %d\nFile: %s\n", courseID, assignID, headerRow, studentIdCol, scoreCol, commentCol, csvFile)
		reader := cmd.InOrStdin()
		fmt.Println(reader)
		os.Exit(1)
		data, err := readCSV(csvFile, ReaderOptions{
			headerRow:    headerRow,
			studentIdCol: studentIdCol,
			scoreCol:     scoreCol,
			commentCol:   commentCol,
		})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error Running command %v\n", err)
			os.Exit(-2)
		}

		builtReqs, err := buildRequests(data, courseID, assignID)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error Running command %v\n", err)
			os.Exit(-2)
		}
		//start to send reqs
		client := &http.Client{}
		batchSize := 20
		var wg sync.WaitGroup
		comm := make(chan ResponseResult, len(builtReqs))
		for i := 0; i < len(builtReqs); i += batchSize {
			end := i + batchSize
			if end > len(builtReqs) {
				end = len(builtReqs)
			}
			wg.Add(1)
			go sendRequests(client, builtReqs[i:end], &comm, &wg)

		}
		//wait for all go routines to finish their batches
		go func() {
			wg.Wait()
			close(comm)
		}()
		failed := 0.0
		success := 0.0
		size := float64(len(builtReqs))
		for response := range comm {
			if response.code >= 400 {
				_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", response.msg)
				failed++
			} else {
				success++
			}
			fmt.Printf("\rTotal: %4.f Success: %.f Failed: %.f  %.2f%%", size, success, failed, ((success+failed)/size)*100)
		}
	}
	if err := rootCmd.Execute(); err != nil {
		logger.Printf("Error Running command %v", err)
		os.Exit(-2)
	}
}
