package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func unzipTests(postBody []byte, outputDir string) {
	zipFile, err := os.Create(path.Join(outputDir, "tests.zip"))
	if err != nil {
		log.Fatal(err)
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			log.Fatal(err)
		}
		err = os.Remove(zipFile.Name())
		if err != nil {
			log.Fatal(err)
		}
	}(zipFile)

	_, err = io.Copy(zipFile, bytes.NewReader(postBody))
	if err != nil {
		log.Fatal(err)
	}

	zipReader, err := zip.OpenReader(zipFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer func(zipReader *zip.ReadCloser) {
		err := zipReader.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(zipReader)

	testDir := path.Join(outputDir, "tests")
	err = os.MkdirAll(testDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range zipReader.File {
		testFileName := path.Join(testDir, f.Name)
		outFile, err := os.OpenFile(testFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Fatal(err)
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			log.Fatal(err)
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func createProblemDir(problemDirName string) {
	if dirExists(problemDirName) {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("About to create directory: '" + problemDirName + "' (y/n)? ")
	input, _ := reader.ReadString('\n')
	input = strings.Trim(strings.ToLower(input), "\r\n ")
	if input != "y" {
		return
	}

	err := os.MkdirAll(problemDirName, 0777)
	if err != nil {
		log.Fatal(err)
	}
}

func downloadTests(csrfToken, testDataUrl string, cookie *http.Cookie, client *http.Client) []byte {
	formData := url.Values{}
	formData.Add("csrf_token", csrfToken)
	formData.Add("download", "true")
	request, err := http.NewRequest("POST", testDataUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	request.AddCookie(cookie)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	postResponse, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	body, err := io.ReadAll(postResponse.Body)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: testgrabber problemNumber cookieValue")
		os.Exit(1)
	}

	problemNum := os.Args[1]
	cookieValue := os.Args[2]

	const baseTestDataUrl string = "https://cses.fi/problemset/tests/"
	testDataUrl := fmt.Sprintf("%s%s", baseTestDataUrl, problemNum)
	cookie := &http.Cookie{
		Name:  "PHPSESSID",
		Value: cookieValue,
	}

	request, err := http.NewRequest("GET", testDataUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	request.AddCookie(cookie)

	client := &http.Client{}
	getResponse, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(getResponse.Body)

	body, err := io.ReadAll(getResponse.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyStr := string(body)
	pattern, _ := regexp.Compile(`<input type="hidden" name="csrf_token" value="(.*)">`)
	match := pattern.FindStringSubmatch(bodyStr)
	if match == nil {
		log.Fatal("Can't find csrf_token")
	}
	csrfToken := match[1]

	// Find the problem category and name
	pattern, _ = regexp.Compile(`(?s)<div class="nav sidebar">\r?\n<h4>([a-zA-Z ]*)</h4>`)
	match = pattern.FindStringSubmatch(bodyStr)
	if match == nil {
		log.Fatal("Can't find problem category")
	}
	problemCategory := strings.ReplaceAll(match[1], " ", "")

	pattern, _ = regexp.Compile(`<h1>([a-zA-Z ]*)</h1>`)
	match = pattern.FindStringSubmatch(bodyStr)
	if match == nil {
		log.Fatal("Can't find problem name")
	}
	problemName := strings.ReplaceAll(match[1], " ", "")

	postBody := downloadTests(csrfToken, testDataUrl, cookie, client)

	problemDir := fmt.Sprintf("../../ProblemSet/%s/%s", problemCategory, problemName)
	createProblemDir(problemDir)

	unzipTests(postBody, problemDir)

	fmt.Println("Done!")
}
