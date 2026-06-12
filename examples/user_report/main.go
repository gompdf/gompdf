package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gompdf/gompdf"
)

type User struct {
	ID        string
	Name      string
	Email     string
	LastLogin string
	Sessions  int
	Status    string
}

type ReportData struct {
	Date               string
	TotalUsers         int
	NewUsers           int
	ActiveUsers        int
	ActiveUsersChange  int
	AvgSessionDuration string
	AvgSessionChange   int
	Users              []User
	CurrentPage        int
	TotalPages         int
}

func generateSampleUsers(count int) []User {
	firstNames := []string{"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "William", "Elizabeth", "David", "Susan", "Richard", "Jessica", "Joseph", "Sarah", "Thomas", "Karen", "Charles", "Nancy"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Jones", "Brown", "Davis", "Miller", "Wilson", "Moore", "Taylor", "Anderson", "Thomas", "Jackson", "White", "Harris", "Martin", "Thompson", "Garcia", "Martinez", "Robinson"}
	domains := []string{"gmail.com", "yahoo.com", "outlook.com", "hotmail.com", "aol.com", "icloud.com", "mail.com"}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	users := make([]User, count)

	for i := 0; i < count; i++ {
		firstName := firstNames[r.Intn(len(firstNames))]
		lastName := lastNames[r.Intn(len(lastNames))]
		domain := domains[r.Intn(len(domains))]
		email := fmt.Sprintf("%s.%s@%s", strings.ToLower(firstName[:4]), strings.ToLower(lastName[:4]), domain)

		daysAgo := r.Intn(30)
		lastLogin := time.Now().AddDate(0, 0, -daysAgo).Format("2006-01-02")

		sessions := r.Intn(100) + 1

		var status string
		statusRoll := r.Intn(10)
		if statusRoll < 7 {
			status = "active"
		} else if statusRoll < 9 {
			status = "inactive"
		} else {
			status = "pending"
		}

		users[i] = User{
			ID:        fmt.Sprintf("U%05d", i+1000),
			Name:      fmt.Sprintf("%s %s", firstName, lastName),
			Email:     email,
			LastLogin: lastLogin,
			Sessions:  sessions,
			Status:    status,
		}
	}

	return users
}

func main() {
	users := generateSampleUsers(100)

	activeUsers := 0
	for _, user := range users {
		if user.Status == "active" {
			activeUsers++
		}
	}

	reportData := ReportData{
		Date:               time.Now().Format("January 2, 2006"),
		TotalUsers:         len(users),
		NewUsers:           15,
		ActiveUsers:        activeUsers,
		ActiveUsersChange:  12,
		AvgSessionDuration: "24m 32s",
		AvgSessionChange:   8,
		Users:              users,
		CurrentPage:        1,
		TotalPages:         3,
	}

	tmpl, err := template.ParseFiles("report.tmpl.html")
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	htmlFile, err := os.Create("report.html")
	if err != nil {
		log.Fatalf("Error creating HTML file: %v", err)
	}
	defer htmlFile.Close()

	err = tmpl.Execute(htmlFile, reportData)
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}
	absPath, err := filepath.Abs("report.html")
	if err != nil {
		log.Fatalf("Error getting absolute path: %v", err)
	}

	fmt.Println("HTML content written to report.html")

	fmt.Println("Starting PDF conversion (tailwind-report)...")

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	// Create PDF with options
	options := gompdf.DefaultOptions()
	options.PageWidth = gompdf.PageSizeLetterWidth
	options.PageHeight = gompdf.PageSizeLetterHeight
	options.PageOrientation = gompdf.PageOrientationLandscape
	options.MarginTop = 36
	options.MarginBottom = 36
	options.MarginLeft = 36
	options.MarginRight = 36
	options.ResourcePaths = []string{currentDir}
	// options.Debug = true
	// Enable debug mode for logging but disable box drawing

	converter := gompdf.NewWithOptions(options)

	outputPath := "report.pdf"
	err = converter.ConvertFile(absPath, outputPath)
	if err != nil {
		log.Fatalf("Error generating PDF: %v", err)
	}

	fileInfo, err := os.Stat("report.pdf")
	if err != nil {
		log.Fatalf("Error getting file info: %v", err)
	}

	fmt.Printf("Tailwind report PDF generated: report.pdf | size=%d bytes\n", fileInfo.Size())
}
