package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

const inputFileExt = ".in"
const outputFileExt = ".out"
const userOutputPrefix = "my_"

type testResult int

const (
	Pass testResult = iota
	Fail
	Timeout
)

var resultName = map[testResult]string{
	Pass:    "Pass",
	Fail:    "Fail",
	Timeout: "Timeout",
}

func (r testResult) String() string {
	return resultName[r]
}

type testReport struct {
	result   testResult
	time     time.Duration
	testName string
}

func newTestReport(testName string) *testReport {
	report := testReport{testName: testName}
	return &report
}

func runTest(command string, testDir string, testFile string, outputDir string, timeout time.Duration) *testReport {
	inFile := path.Join(testDir, testFile)
	input, err := os.Open(inFile)
	if err != nil {
		log.Fatal(err)
	}
	defer func(input *os.File) {
		err := input.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(input)

	testName := strings.TrimSuffix(testFile, inputFileExt)
	report := newTestReport(testName)

	parts := strings.Split(command, " ")
	runCmd := exec.Command(parts[0])
	runCmd.Stdin = input
	runCmd.Args = parts

	type cmdResult struct {
		out []byte
		err error
	}

	cmdDone := make(chan cmdResult, 1)
	go func() {
		start := time.Now()
		out, testErr := runCmd.Output()
		if testErr != nil {
			log.Fatal(testErr)
		}
		report.time = time.Since(start)
		cmdDone <- cmdResult{out, testErr}
	}()

	var output cmdResult
	select {
	case <-time.After(timeout):
		_ = runCmd.Process.Kill()
		report.result = Timeout
		report.time = timeout
	case output = <-cmdDone:
		// Use consistent line endings to avoid hash compare failures
		output.out = bytes.ReplaceAll(output.out, []byte("\r\n"), []byte("\n"))
		if len(outputDir) > 0 {
			outfile := path.Join(outputDir, userOutputPrefix+testName+outputFileExt)
			err = os.WriteFile(outfile, output.out, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}

		myOutHash := sha256.Sum256(output.out)

		testOutput := path.Join(testDir, testName+outputFileExt)
		comparisonOutput, err := os.Open(testOutput)
		if err != nil {
			log.Fatal(err)
		}

		hasher := sha256.New()
		_, err = io.Copy(hasher, comparisonOutput)
		testOutHash := hasher.Sum(nil)

		matchingOutput := bytes.Equal(myOutHash[:], testOutHash[:])
		if matchingOutput {
			report.result = Pass
		} else {
			report.result = Fail
		}
	}

	return report
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: runner command")
		os.Exit(1)
	}

	const timeout = 2 * time.Second

	const relativeRoot = "../../ProblemSet/"
	command := path.Join(relativeRoot, os.Args[1])
	testDir := path.Join(path.Dir(path.Dir(command)), "tests")
	outputDir := path.Join(path.Dir(command), "output")
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	entries, err := os.ReadDir(testDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%-10s%-8s%s\n", "Test Name", "Result", "Duration")

	var wg sync.WaitGroup

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), inputFileExt) {
			continue
		}

		wg.Go(func() {
			result := runTest(command, testDir, entry.Name(), outputDir, timeout)
			fmt.Printf("%-10s%-8s%s\n", result.testName, result.result.String(), result.time)
		})
	}
	wg.Wait()
}
