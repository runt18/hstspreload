package hstspreload

import (
	"fmt"
	"net/http"
	"testing"
)

const (
	headerStringsShouldBeEqual = `Did not receive expected header.
			Actual: "%v"
			Expected: "%v"`
)

/******** Examples. ********/

func ExamplePreloadableResponse() {
	resp, err := http.Get("localhost:8080")
	if err != nil {
		header, issues := PreloadableResponse(resp)
		if header != nil {
			fmt.Printf("Header: %s", *header)
		}
		fmt.Printf("Issues: %v", issues)
	}
}

/******** Response tests. ********/

var responseTests = []struct {
	function       func(resp *http.Response) (header *string, issues Issues)
	description    string
	hstsHeaders    []string
	expectedIssues Issues
}{

	/******** PreloadableResponse() ********/

	{
		PreloadableResponse,
		"good header",
		[]string{"max-age=10886400; includeSubDomains; preload"},
		Issues{},
	},
	{
		PreloadableResponse,
		"missing preload",
		[]string{"max-age=10886400; includeSubDomains"},
		Issues{Errors: []Issue{Issue{Code: "header.preloadable.preload.missing"}}},
	},
	{
		PreloadableResponse,
		"missing includeSubDomains",
		[]string{"preload; max-age=10886400"},
		Issues{Errors: []Issue{Issue{Code: "header.preloadable.include_sub_domains.missing"}}},
	},
	{
		PreloadableResponse,
		"single header, multiple errors",
		[]string{"includeSubDomains; max-age=100"},
		Issues{
			Errors: []Issue{
				Issue{Code: "header.preloadable.preload.missing"},
				Issue{
					Code:    "header.preloadable.max_age.too_low",
					Message: "The max-age must be at least 10886400 seconds (== 18 weeks), but the header currently only has max-age=100.",
				},
			},
		},
	},
	{
		PreloadableResponse,
		"empty header",
		[]string{""},
		Issues{
			Errors: []Issue{
				Issue{Code: "header.preloadable.include_sub_domains.missing"},
				Issue{Code: "header.preloadable.preload.missing"},
				Issue{Code: "header.preloadable.max_age.missing"},
			},
			Warnings: []Issue{Issue{Code: "header.parse.empty"}},
		},
	},
	{
		PreloadableResponse,
		"missing header",
		[]string{},
		Issues{Errors: []Issue{Issue{Code: "response.no_header"}}},
	},
	{
		PreloadableResponse,
		"multiple headers",
		[]string{"max-age=10", "max-age=20", "max-age=30"},
		Issues{Errors: []Issue{Issue{Code: "response.multiple_headers"}}},
	},

	/******** RemovableResponse() ********/

	{
		RemovableResponse,
		"no preload",
		[]string{"max-age=15768000; includeSubDomains"},
		Issues{},
	},
	{
		RemovableResponse,
		"preload present",
		[]string{"max-age=15768000; includeSubDomains; preload"},
		Issues{Errors: []Issue{Issue{Code: "header.removable.contains.preload"}}},
	},
	{
		RemovableResponse,
		"preload only",
		[]string{"preload"},
		Issues{
			Errors: []Issue{
				Issue{Code: "header.removable.contains.preload"},
				Issue{Code: "header.removable.missing.max_age"},
			},
		},
	},
}

func TestPreloabableResponseAndRemovableResponse(t *testing.T) {
	for _, tt := range responseTests {

		resp := &http.Response{}
		resp.Header = http.Header{}

		key := http.CanonicalHeaderKey("Strict-Transport-Security")
		for _, h := range tt.hstsHeaders {
			resp.Header.Add(key, h)
		}

		header, issues := tt.function(resp)

		if len(tt.hstsHeaders) == 1 {
			if header == nil {
				t.Errorf("[%s] Did not receive exactly one HSTS header", tt.description)
			} else if *header != tt.hstsHeaders[0] {
				t.Errorf("[%s] "+headerStringsShouldBeEqual, tt.description, *header, tt.hstsHeaders[0])
			}
		} else {
			if header != nil {
				t.Errorf("[%s] Did not expect a header, but received `%s`", tt.description, *header)
			}
		}

		if !issuesMatchExpected(issues, tt.expectedIssues) {
			t.Errorf("[%s] "+issuesShouldMatch, tt.description, issues, tt.expectedIssues)
		}
	}
}
