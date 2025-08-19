package testing

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"time"
)

// TestReport represents a comprehensive test report
type TestReport struct {
	Timestamp    time.Time        `json:"timestamp" xml:"timestamp,attr"`
	Duration     time.Duration    `json:"duration" xml:"duration,attr"`
	TotalTests   int              `json:"total_tests" xml:"total_tests,attr"`
	PassedTests  int              `json:"passed_tests" xml:"passed_tests,attr"`
	FailedTests  int              `json:"failed_tests" xml:"failed_tests,attr"`
	SkippedTests int              `json:"skipped_tests" xml:"skipped_tests,attr"`
	Suites       []*TestSuite     `json:"suites" xml:"suite"`
	Summary      *TestSummary     `json:"summary" xml:"summary"`
	Environment  *TestEnvironment `json:"environment" xml:"environment"`
}

// TestSuite represents a collection of related tests
type TestSuite struct {
	Name       string          `json:"name" xml:"name,attr"`
	Duration   time.Duration   `json:"duration" xml:"duration,attr"`
	Tests      []*TestCase     `json:"tests" xml:"test"`
	Passed     int             `json:"passed" xml:"passed,attr"`
	Failed     int             `json:"failed" xml:"failed,attr"`
	Skipped    int             `json:"skipped" xml:"skipped,attr"`
	Errors     []*TestError    `json:"errors,omitempty" xml:"error,omitempty"`
	Properties []*TestProperty `json:"properties,omitempty" xml:"property,omitempty"`
}

// TestCase represents a single test case
type TestCase struct {
	Name        string          `json:"name" xml:"name,attr"`
	ClassName   string          `json:"class_name" xml:"classname,attr"`
	Duration    time.Duration   `json:"duration" xml:"time,attr"`
	Status      TestStatus      `json:"status" xml:"status,attr"`
	Output      string          `json:"output,omitempty" xml:"system-out,omitempty"`
	ErrorOutput string          `json:"error_output,omitempty" xml:"system-err,omitempty"`
	Failure     *TestFailure    `json:"failure,omitempty" xml:"failure,omitempty"`
	Error       *TestError      `json:"error,omitempty" xml:"error,omitempty"`
	Properties  []*TestProperty `json:"properties,omitempty" xml:"property,omitempty"`
}

// TestStatus represents the status of a test
type TestStatus string

const (
	TestStatusPassed  TestStatus = "passed"
	TestStatusFailed  TestStatus = "failed"
	TestStatusSkipped TestStatus = "skipped"
	TestStatusError   TestStatus = "error"
)

// TestFailure represents a test failure
type TestFailure struct {
	Message string `json:"message" xml:"message,attr"`
	Type    string `json:"type" xml:"type,attr"`
	Content string `json:"content" xml:",chardata"`
}

// TestError represents a test error
type TestError struct {
	Message string `json:"message" xml:"message,attr"`
	Type    string `json:"type" xml:"type,attr"`
	Content string `json:"content" xml:",chardata"`
}

// TestProperty represents a key-value property
type TestProperty struct {
	Name  string `json:"name" xml:"name,attr"`
	Value string `json:"value" xml:"value,attr"`
}

// TestSummary provides summary statistics
type TestSummary struct {
	CompilationTime time.Duration `json:"compilation_time" xml:"compilation_time,attr"`
	ExecutionTime   time.Duration `json:"execution_time" xml:"execution_time,attr"`
	AverageTestTime time.Duration `json:"average_test_time" xml:"average_test_time,attr"`
	FastestTest     string        `json:"fastest_test" xml:"fastest_test,attr"`
	SlowestTest     string        `json:"slowest_test" xml:"slowest_test,attr"`
	MemoryUsage     int64         `json:"memory_usage" xml:"memory_usage,attr"`
	CoveragePercent float64       `json:"coverage_percent" xml:"coverage_percent,attr"`
}

// TestEnvironment captures environment information
type TestEnvironment struct {
	CompilerVersion string            `json:"compiler_version" xml:"compiler_version,attr"`
	Platform        string            `json:"platform" xml:"platform,attr"`
	Architecture    string            `json:"architecture" xml:"architecture,attr"`
	GoVersion       string            `json:"go_version" xml:"go_version,attr"`
	Variables       map[string]string `json:"variables" xml:"variable"`
}

// ReportGenerator generates test reports in various formats
type ReportGenerator struct {
	report *TestReport
}

// NewReportGenerator creates a new report generator
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		report: &TestReport{
			Timestamp: time.Now(),
			Suites:    make([]*TestSuite, 0),
		},
	}
}

// AddSuite adds a test suite to the report
func (rg *ReportGenerator) AddSuite(suite *TestSuite) {
	rg.report.Suites = append(rg.report.Suites, suite)
	rg.report.TotalTests += len(suite.Tests)
	rg.report.PassedTests += suite.Passed
	rg.report.FailedTests += suite.Failed
	rg.report.SkippedTests += suite.Skipped
}

// SetEnvironment sets the test environment information
func (rg *ReportGenerator) SetEnvironment(env *TestEnvironment) {
	rg.report.Environment = env
}

// SetSummary sets the test summary information
func (rg *ReportGenerator) SetSummary(summary *TestSummary) {
	rg.report.Summary = summary
}

// Finalize finalizes the report with calculated metrics
func (rg *ReportGenerator) Finalize() {
	rg.report.Duration = time.Since(rg.report.Timestamp)

	if rg.report.Summary == nil {
		rg.report.Summary = &TestSummary{}
	}

	// Calculate average test time
	if rg.report.TotalTests > 0 {
		totalTestTime := time.Duration(0)
		var fastestTime, slowestTime time.Duration
		var fastestTest, slowestTest string

		first := true
		for _, suite := range rg.report.Suites {
			for _, test := range suite.Tests {
				totalTestTime += test.Duration

				if first || test.Duration < fastestTime {
					fastestTime = test.Duration
					fastestTest = test.Name
				}

				if first || test.Duration > slowestTime {
					slowestTime = test.Duration
					slowestTest = test.Name
				}

				first = false
			}
		}

		rg.report.Summary.AverageTestTime = totalTestTime / time.Duration(rg.report.TotalTests)
		rg.report.Summary.FastestTest = fastestTest
		rg.report.Summary.SlowestTest = slowestTest
	}
}

// GenerateJSON generates a JSON report
func (rg *ReportGenerator) GenerateJSON(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rg.report)
}

// GenerateXML generates an XML report (JUnit format)
func (rg *ReportGenerator) GenerateXML(writer io.Writer) error {
	// XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Root element
	start := xml.StartElement{
		Name: xml.Name{Local: "testsuites"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "name"}, Value: "Orizon Compiler Tests"},
			{Name: xml.Name{Local: "tests"}, Value: fmt.Sprintf("%d", rg.report.TotalTests)},
			{Name: xml.Name{Local: "failures"}, Value: fmt.Sprintf("%d", rg.report.FailedTests)},
			{Name: xml.Name{Local: "errors"}, Value: "0"},
			{Name: xml.Name{Local: "time"}, Value: fmt.Sprintf("%.3f", rg.report.Duration.Seconds())},
			{Name: xml.Name{Local: "timestamp"}, Value: rg.report.Timestamp.Format(time.RFC3339)},
		},
	}

	if err := encoder.EncodeToken(start); err != nil {
		return err
	}

	// Encode each test suite
	for _, suite := range rg.report.Suites {
		if err := encoder.Encode(suite); err != nil {
			return err
		}
	}

	// Close root element
	if err := encoder.EncodeToken(start.End()); err != nil {
		return err
	}

	return encoder.Flush()
}

// GenerateHTML generates an HTML report
func (rg *ReportGenerator) GenerateHTML(writer io.Writer) error {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Orizon Compiler Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f4f4f4; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 30px; }
        .metric { background-color: #e8f4f8; padding: 15px; border-radius: 5px; text-align: center; }
        .metric h3 { margin: 0; color: #333; }
        .metric .value { font-size: 24px; font-weight: bold; color: #2196F3; }
        .suite { margin-bottom: 30px; border: 1px solid #ddd; border-radius: 5px; }
        .suite-header { background-color: #f8f9fa; padding: 15px; border-bottom: 1px solid #ddd; }
        .suite-name { font-size: 18px; font-weight: bold; }
        .test-list { padding: 0; margin: 0; list-style: none; }
        .test-item { padding: 10px 15px; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; align-items: center; }
        .test-item:last-child { border-bottom: none; }
        .test-name { font-weight: 500; }
        .test-status { padding: 4px 8px; border-radius: 3px; font-size: 12px; font-weight: bold; }
        .status-passed { background-color: #d4edda; color: #155724; }
        .status-failed { background-color: #f8d7da; color: #721c24; }
        .status-skipped { background-color: #fff3cd; color: #856404; }
        .test-duration { color: #666; font-size: 12px; }
        .error-details { background-color: #f8f9fa; padding: 10px; margin-top: 10px; border-left: 4px solid #dc3545; }
        .progress-bar { width: 100%; height: 20px; background-color: #e9ecef; border-radius: 10px; overflow: hidden; }
        .progress-fill { height: 100%; background-color: #28a745; transition: width 0.3s ease; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Orizon Compiler Test Report</h1>
        <p>Generated on ` + rg.report.Timestamp.Format("January 2, 2006 at 3:04 PM") + `</p>
        <p>Total Duration: ` + rg.report.Duration.String() + `</p>
    </div>

    <div class="summary">
        <div class="metric">
            <h3>Total Tests</h3>
            <div class="value">` + fmt.Sprintf("%d", rg.report.TotalTests) + `</div>
        </div>
        <div class="metric">
            <h3>Passed</h3>
            <div class="value" style="color: #28a745;">` + fmt.Sprintf("%d", rg.report.PassedTests) + `</div>
        </div>
        <div class="metric">
            <h3>Failed</h3>
            <div class="value" style="color: #dc3545;">` + fmt.Sprintf("%d", rg.report.FailedTests) + `</div>
        </div>
        <div class="metric">
            <h3>Success Rate</h3>
            <div class="value">` + fmt.Sprintf("%.1f%%", float64(rg.report.PassedTests)/float64(rg.report.TotalTests)*100) + `</div>
        </div>
    </div>

    <div class="progress-bar">
        <div class="progress-fill" style="width: ` + fmt.Sprintf("%.1f%%", float64(rg.report.PassedTests)/float64(rg.report.TotalTests)*100) + `;"></div>
    </div>
    <br>
`

	// Add test suites
	for _, suite := range rg.report.Suites {
		html += `
    <div class="suite">
        <div class="suite-header">
            <div class="suite-name">` + suite.Name + `</div>
            <div>Duration: ` + suite.Duration.String() + ` | Passed: ` + fmt.Sprintf("%d", suite.Passed) + ` | Failed: ` + fmt.Sprintf("%d", suite.Failed) + `</div>
        </div>
        <ul class="test-list">`

		for _, test := range suite.Tests {
			statusClass := "status-" + string(test.Status)
			html += `
            <li class="test-item">
                <div>
                    <span class="test-name">` + test.Name + `</span>
                    <span class="test-status ` + statusClass + `">` + string(test.Status) + `</span>
                </div>
                <span class="test-duration">` + test.Duration.String() + `</span>`

			if test.Failure != nil || test.Error != nil {
				var errorMsg string
				if test.Failure != nil {
					errorMsg = test.Failure.Message
				} else if test.Error != nil {
					errorMsg = test.Error.Message
				}
				html += `
                <div class="error-details">` + errorMsg + `</div>`
			}

			html += `
            </li>`
		}

		html += `
        </ul>
    </div>`
	}

	// Close HTML
	html += `
</body>
</html>`

	_, err := writer.Write([]byte(html))
	return err
}

// SaveToFile saves the report to a file in the specified format
func (rg *ReportGenerator) SaveToFile(filename string, format string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	switch format {
	case "json":
		return rg.GenerateJSON(file)
	case "xml":
		return rg.GenerateXML(file)
	case "html":
		return rg.GenerateHTML(file)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ConvertTestResultToCase converts a TestResult to a TestCase
func ConvertTestResultToCase(test *CompilerTest, result *TestResult) *TestCase {
	var status TestStatus
	var failure *TestFailure
	var testError *TestError

	if result.Success {
		status = TestStatusPassed
	} else {
		status = TestStatusFailed
		if result.Error != nil {
			failure = &TestFailure{
				Message: result.Error.Error(),
				Type:    "CompilationError",
				Content: result.ErrorOutput,
			}
		}
	}

	return &TestCase{
		Name:        test.Name,
		ClassName:   "CompilerTest",
		Duration:    result.Duration,
		Status:      status,
		Output:      result.Output,
		ErrorOutput: result.ErrorOutput,
		Failure:     failure,
		Error:       testError,
	}
}
