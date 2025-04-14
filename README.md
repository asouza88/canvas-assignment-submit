# canvas-assignment-submit

A simple Go CLI tool that allows you to upload assignment feedback and grades to [Canvas LMS](https://www.instructure.com/canvas) using a CSV file.

## 🚀 Features

-   Uploads comments and scores to Canvas assignments
-   Command-line interface for fast usage
-   Supports cross-platform builds for Windows, Linux, and macOS
-   Configurable via command-line arguments

## 🛠 Build

This project includes two build scripts for cross-compiling:

-   `build.sh`: Bash script for Linux/macOS
-   `build.ps1`: PowerShell script for Windows

### Example (Linux/macOS):

```bash
./build.sh linux         # Builds for Linux
./build.sh windows       # Builds for Windows
./build.sh mac-arm64     # Builds for macOS (Apple Silicon)
./build.sh mac-amd64     # Builds for macOS (Intel)
```

### Example (Windows):

```bash
./build.ps1 linux        # Builds for Linux
./build.ps1 windows       # Builds for Windows
./build.ps1 mac-arm64     # Builds for macOS (Apple Silicon)
./build.ps1 mac-amd64     # Builds for macOS (Intel)
```

### Go command

```bash
go run . -c cavnasid -a assignmentid csvfile
```

## Environment Variables

The program expects to have 2 environment variables set before execution.
| Variable | Description |
| -------- | ----------------------------------- |
| CANVAS_API | Canvas base domain for your institution. For example, https://school.instructure.com/api/v1 |
| CANVAS_TOKEN | Token authorizing access to Canvas LMS REST API. More infor can be found [here](https://canvas.instructure.com/doc/api/) |

## Example usage

```bash
cal -c 12345 -a 67890 csvfile
```

### Arguments

| Flag | Type | Description                         | Default  |
| ---- | ---- | ----------------------------------- | -------- |
| -c   | int  | Canvas Course ID                    | required |
| -a   | int  | Canvas Assignment ID                | required |
| -h   | int  | Header row index in the CSV         | 0        |
| -i   | int  | Column index for Student ID         | 0        |
| -s   | int  | Column index for Assignment Score   | 1        |
| -t   | int  | Column index for Assignment Comment | 2        |
