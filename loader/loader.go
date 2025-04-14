package loader

import (
	"encoding/csv"
	"fmt"
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

/*
All reader options are assumed defaults,
not reading from stdin for now as it doesnt work as expected
below code is ok but the initial startup is broken 4-14-2025
*/
// func readCSVDataFromSTDIN() (map[string]AssignmentFeedback, error) {
// 	scanner := bufio.NewScanner(os.Stdin)
// 	headerRow := scanner.Text()
// 	scanner.Scan()//move past header row
// 	attendData := make(map[string]AssignmentFeedback)
// 	headerItems := strings.Split(headerRow, ",")
// 	for scanner.Scan() {
// 		lineItems := strings.Split(scanner.Text(), ",")
// 		if len(lineItems) < len(headerItems) {
// 			log.Fatalln("invalid data format, some rows are missing data")
// 		}
// 		attendData[lineItems[0]] = AssignmentFeedback{
// 			studentId: lineItems[0],
// 			score:     lineItems[1],
// 			comment:   lineItems[2],
// 		}
// 	}
// 	if err := scanner.Err(); err != nil {
// 		return nil, err
// 	}
// 	return attendData, nil
// }

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
	if sizeOfRow < options.scoreCol || sizeOfRow < 0 || sizeOfRow < options.commentCol {
		return nil, fmt.Errorf("invalid index used for given reader options. Row length %d , Options: %v", sizeOfRow, options)
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
		attendData[record[0]] = AssignmentFeedback{
			studentId: record[0],
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

// LoadScores will upload assignment feedback to Canvas LMS using Canvas's REST API
// the program will lok for CANVAS_DOMAIN and CANVAS_TOKEN evironment variables.
func LoadScores(courseID int,
	assignmentID int,
	headerRow int,
	studentIdCol int,
	scoreCol int,
	commentCol int,
	csvFile string) {
	


	var data map[string]AssignmentFeedback
	var err error
	data, err = readCSV(csvFile, ReaderOptions{
		headerRow:    headerRow,
		studentIdCol: studentIdCol,
		scoreCol:     scoreCol,
		commentCol:   commentCol,
	})
	if err != nil {
		log.Fatalln("failed reading from csv file ", csvFile)
	}

	fmt.Printf("Course Id: %d\nAssign Id: %d\nHeader Row index: %d\nStudent Id Col: %d\nScore Col: %d\nComment Col: %d\nCSVFile: %s\n", courseID, assignmentID, headerRow, studentIdCol, scoreCol, commentCol, csvFile)

	builtReqs, err := buildRequests(data, courseID, assignmentID)
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
