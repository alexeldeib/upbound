package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/alexeldeib/upbound/pkg/handlers"
	"github.com/alexeldeib/upbound/pkg/types"
	"github.com/sirupsen/logrus"
)

var server handlers.Server

func TestMain(m *testing.M) {
	server = handlers.NewServer()
	logrus.SetOutput(ioutil.Discard)
	m.Run()
}

func TestSimpleCreate(t *testing.T) {
	yaml := `title: Valid App 1
version: 0.0.1
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
- name: secondmaintainer app1
  email: secondmaintainer@gmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: |
  ### Interesting Title
  Some application content, and description`

	rr := execute(yaml, "PUT", "/create", server.Create, t)

	// Should succeed
	status := rr.Code
	equals(t, http.StatusCreated, status)

	// Check the response body is what we expect.
	expected := ""
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestConflictCreate(t *testing.T) {
	yaml := `title: Valid App 1
version: 0.0.1
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
- name: secondmaintainer app1
  email: secondmaintainer@gmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: |
  ### Interesting Title
  Some application content, and description`

	// Executing twice should cause conflict
	rr := execute(yaml, "PUT", "/create", server.Create, t)
	rr = execute(yaml, "PUT", "/create", server.Create, t)

	// Validate we conflicted
	status := rr.Code
	equals(t, http.StatusConflict, status)

	expected := "An application with title Valid App 1 already exists, please use a unique title."
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestMissingVersion(t *testing.T) {
	yaml := `title: App w/ missing version
maintainers:
- name: first last
  email: email@hotmail.com
- name: first last
  email: email@gmail.com
company: Company Inc.
website: https://website.com
source: https://github.com/company/repo
license: Apache-2.0
description: |
  ### blob of markdown
  More markdown`

	rr := execute(yaml, "PUT", "/create", server.Create, t)

	// Should succeed
	status := rr.Code
	equals(t, http.StatusBadRequest, status)

	// Check the response body is what we expect.
	expected := "Failed to validate input of the following parameters:\nApplicationMetadata.Version has invalid value \n"
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestInvalidEmail(t *testing.T) {
	yaml := `title: App w/ Invalid maintainer email
version: 1.0.1
maintainers:
- name: Firstname Lastname
  email: apptwohotmail.com
company: Upbound Inc.
website: https://upbound.io
source: https://github.com/upbound/repo
license: Apache-2.0
description: |
  ### blob of markdown
  More markdown`

	rr := execute(yaml, "PUT", "/create", server.Create, t)

	status := rr.Code
	equals(t, http.StatusBadRequest, status)

	expected := "Failed to validate input of the following parameters:\nApplicationMetadata.Maintainers[0].Email has invalid value apptwohotmail.com\n"
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestMalformedYaml(t *testing.T) {
	yaml := `title: App w/ Invalid maintainer email
version: 1.0.1
 maintainers:
- name: Firstname Lastname
 email: app@hotmail.com
company: Upbound Inc.
 website: https://upbound.io
source: https://github.com/upbound/repo
license: Apache-2.0
description: |
### blob of markdown
More markdown`

	rr := execute(yaml, "PUT", "/create", server.Create, t)

	status := rr.Code
	equals(t, http.StatusBadRequest, status)

	expected := "Failed to parse YAML input. This likely indicates malformed request body. Verify the payload fields and parameter types are correct.\n"
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestWrongParameterTypes(t *testing.T) {
	yaml := `title: Valid App 1
version: [ 0.0.1, 0.0.2 ]
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: Another equally interesting description`

	rr := execute(yaml, "PUT", "/create", server.Create, t)

	status := rr.Code
	equals(t, http.StatusBadRequest, status)

	expected := "Failed to parse YAML input. This likely indicates malformed request body. Verify the payload fields and parameter types are correct.\n"
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestSimpleSearch(t *testing.T) {
	yaml := `title: Valid App 1
version: 0.0.1
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
- name: secondmaintainer app1
  email: secondmaintainer@gmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: |
  ### Interesting Title
  Some application content, and description`

	expected := `- title: Valid App 1
  version: 0.0.1
  maintainers:
  - name: firstmaintainer app1
    email: firstmaintainer@hotmail.com
  - name: secondmaintainer app1
    email: secondmaintainer@gmail.com
  company: Random Inc.
  website: https://website.com
  source: https://github.com/random/repo
  license: Apache-2.0
  description: |-
    ### Interesting Title
    Some application content, and description` + "\n"

	rr := execute(yaml, "PUT", "/create", server.Create, t)
	rr = execute(yaml, "POST", "/search", server.Search, t)

	// Should succeed with a single match returned
	status := rr.Code
	equals(t, http.StatusOK, status)

	// Response body should match original input precisely.
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestSearchForTwoApps(t *testing.T) {
	appOne := `title: Valid App 1
version: 0.0.1
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: A really cool app.`

	appTwo := `title: Valid App 2
version: 0.0.1
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: A really cool app.`

	expected := `- title: Valid App 1
  version: 0.0.1
  maintainers:
  - name: firstmaintainer app1
    email: firstmaintainer@hotmail.com
  company: Random Inc.
  website: https://website.com
  source: https://github.com/random/repo
  license: Apache-2.0
  description: A really cool app.
- title: Valid App 2
  version: 0.0.1
  maintainers:
  - name: firstmaintainer app1
    email: firstmaintainer@hotmail.com
  company: Random Inc.
  website: https://website.com
  source: https://github.com/random/repo
  license: Apache-2.0
  description: A really cool app.` + "\n"

	query := `version: 0.0.1`

	rr := execute(appOne, "PUT", "/create", server.Create, t)
	rr = execute(appTwo, "PUT", "/create", server.Create, t)
	rr = execute(query, "POST", "/search", server.Search, t)

	status := rr.Code
	equals(t, http.StatusOK, status)
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestShouldNotFindApp(t *testing.T) {
	yaml := `title: Valid App 1
version: 0.0.1
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: A really cool app.`

	expected := "[]\n"

	query := `version: 0.0.2`

	rr := execute(yaml, "PUT", "/create", server.Create, t)
	rr = execute(query, "POST", "/search", server.Search, t)

	status := rr.Code
	equals(t, http.StatusOK, status)
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestFailWithBadParameter(t *testing.T) {
	yaml := `title: Valid App 1
version: 0.0.1
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: A really cool app.`

	expected := "Failed to parse YAML input. This likely indicates malformed request body. Verify the payload fields and parameter types are correct.\n"

	query := `version: [ 0.0.2 ]`

	rr := execute(yaml, "PUT", "/create", server.Create, t)
	rr = execute(query, "POST", "/search", server.Search, t)

	status := rr.Code
	equals(t, http.StatusBadRequest, status)
	equals(t, expected, rr.Body.String())

	cleanup()
}

func TestMissingEmail(t *testing.T) {
	yaml := `title: Valid App 1
version: 0.0.1
maintainers:
- name: firstmaintainer app1
  email: firstmaintainer@hotmail.com
- name: secondmaintainer app1
  email: secondmaintainer@gmail.com
company: Random Inc.
website: https://website.com
source: https://github.com/random/repo
license: Apache-2.0
description: |
  ### Interesting Title
  Some application content, and description`

	expected := `- title: Valid App 1
  version: 0.0.1
  maintainers:
  - name: firstmaintainer app1
    email: firstmaintainer@hotmail.com
  - name: secondmaintainer app1
    email: secondmaintainer@gmail.com
  company: Random Inc.
  website: https://website.com
  source: https://github.com/random/repo
  license: Apache-2.0
  description: |-
    ### Interesting Title
    Some application content, and description` + "\n"

	search := `maintainers:
- name: firstmaintainer app1`

	rr := execute(yaml, "PUT", "/create", server.Create, t)
	rr = execute(search, "POST", "/search", server.Search, t)

	status := rr.Code
	equals(t, http.StatusOK, status)

	equals(t, expected, rr.Body.String())

	cleanup()
}

// execute assists generating HTTP requests for testing purposes.
func execute(yaml string, method string, endpoint string, f func(http.ResponseWriter, *http.Request), t *testing.T) *httptest.ResponseRecorder {
	// Read data, create a request manually, instantiate recording apparatus.
	data := strings.NewReader(yaml)
	req, err := http.NewRequest(method, endpoint, data)
	ok(t, err)
	rr := httptest.NewRecorder()

	// Create handler and process request
	handler := http.HandlerFunc(f)
	handler.ServeHTTP(rr, req)

	return rr
}

// cleanup clears stored application metadata on the server in between test runs.
func cleanup() {
	server.Applications = []*types.ApplicationMetadata{}
}

// FUNCTIONS BELOW THIS LINE COURTESTY OF https://github.com/benbjohnson/testing
// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
