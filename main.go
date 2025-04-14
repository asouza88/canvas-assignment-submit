package main

import (
	"gocanvas/loader/loader"
	"flag"
	"os"
	"fmt"
)

func main()  {
	var courseID int
	var assignmentID int
	var headerRow int
	var studentIdCol int
	var scoreCol int
	var commentCol int
	var csvFile string
	
	flag.IntVar(&courseID, "c", -1, "Canvas Course ID")
	flag.IntVar(&assignmentID, "a", -1, "Canvas Assignment ID")
	flag.IntVar(&headerRow, "h", 0, "Header row index in CSV")
	flag.IntVar(&studentIdCol, "i", 0, "Column Index for student ID")
	flag.IntVar(&scoreCol, "s", 1, "Column Index for Assignment Score")
	flag.IntVar(&commentCol, "t", 2, "Column index for Comment for Assignment")
	
	flag.Parse()
	
	if courseID < 0 {
		fmt.Println("Positional Argument -c, Canvas CourseID is missing.")
		flag.Usage()
		os.Exit(-1)
	}
	
	if assignmentID < 0 {
		fmt.Println("Positional Argument -a, Canvas AssignmentID is missing.")
		flag.Usage()
		os.Exit(-1)
	}

	if len(flag.Args()) > 0 {
		//csv should be first element in return of flag.Args()
		csvFile = flag.Arg(0)
	} else {
		fmt.Println("Required Argument csvfile is missing OR csv formatted data piped into stdin")
		flag.Usage()
		os.Exit(-2)
	}
	
	loader.LoadScores(courseID,assignmentID,headerRow,studentIdCol,scoreCol,commentCol,csvFile)
}