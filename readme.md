# Upbound Project Log

## Initial Impression

Breaking down the assignment, it can map fairly closely onto a few component pieces.

- Server, probably with a PUT/POST endpoint for creating application metadata and a GET endpoint for searching it using query parameters. Seems safe to stick with `net/http`.
- Parsing of YAML input. I will defer this to an external dependency, probably [go-yaml](https://github.com/go-yaml/yaml).
- Input validation. Given the validation appears fairly simple on first glance, I could manage this without any additional dependencies. Otherwise I will likely defer this to something like [go-validator](https://github.com/go-playground/validator).
- Simple logging for my own debugging and sanity. The usual `log` or `glog` would be plenty here. I may instead use the opportunity to familiarize myself with [logrus](https://github.com/sirupsen/logrus)
- Struct(s) to define application metadata shape.
- An implementation of filtering/searching based on query parameters for the get endpoint. Not sure on first pass how I would like to achieve this. 

## First Steps

I also took this project as an opportunity to improve my knowledge of conventions in Go programming. I found [golang-standard/project-layout](https://github.com/golang-standards/project-layout) which gave me a nice rundown of project structuring in Go. I like the `cmd` and `pkg` conventions in Go. I oped to include a `docs` directory, where this file will (hopefully!) end up. 

Here's how I set up the project to get started:

```
.
├── cmd
│   └── upbound-app
│       └── main.go
├── docs
│   └── readme.md
├── go.mod
├── go.sum
└── pkg
    └── types.go

4 directories, 5 files
```

##  Types

After I scaffolded a basic application structure, I began mapping the example input into Go structs.  These are fairly simple:

```go
type Maintainer struct {
	Name  string
	Email string
}

type ApplicationMetadata struct {
	Title       string
	Version     string
	Maintainers []Maintainer
	Company     string
	Website     string
	Source      string
	License     string
	Description string
}
```

## Go Module Woes

With my basic types scaffolded out, I wanted to test out my import paths using modules. I dabbled previously using go modules for simple utility applications which had < 5 files in a flat directory. I found out standard layouts using `cmd` directory don't map very well to go modules. Seems like go.mod needs to be a peer to the main package. I eliminated the `cmd` dir and moved my `main.go` as a peer to `go.mod`. 

With this in mind, I would probably further restructure this application as would be appropriate. I didn't find great resources on structuring multiple packages inside a single module, so I stuck with using my `pkg` directory (all of this could be in `internal`, given the nature of the assignment).

After hitting this snag, my new project structure looks like this (including a new file for me to log time on the project):

```
.
├── docs
│   ├── readme.md
│   └── time.md
├── go.mod
├── main.go
└── pkg
    └── types
        └── types.go

3 directories, 5 files
```

## Route Handlers

With most of the basic logic set up I created some basic route handlers:

```go
func main() {
	log.SetOutput(os.Stdout)
	http.HandleFunc("/create", create)
	http.HandleFunc("/search", search)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf(err)
	}
}

func create(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Please use a PUT request to create an application.", http.StatusBadRequest)
		return
	}
}

func search(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Please use a GET request to search for an application.", http.StatusBadRequest)
		return
	}
}
```

## Validation Annotations

For my initial implementation I decided to keep things easy and use annotations and an external dependency to manage input validation:

```go
// Maintainer a single maintainer's personal information.
type Maintainer struct {
	Name  string `validate:required`
	Email string `validate:required,email`
}
```

## Parsing Input

Now we need to actually accept and parse input from users! I started extending the route handler for `/create`:

```go
func create(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Please use a PUT request to create an application.", http.StatusBadRequest)
		return
	}
	// Read in body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body of request", http.StatusInternalServerError)
	}
	// Try to parse the metadata content
	metadata := types.ApplicationMetadata{}
	err = yaml.Unmarshal(body, &metadata)
	if err != nil {
		http.Error(w, "Failed to parse YAML input. This likely indicates malformed request body.", http.StatusBadRequest)
	}
}
```

## Struct Validation Struggles

I opted to use `go-playground/validator` to do my input validation. I hit some snags here that almost pushed me to use per-variable validation, manually checking the errors each time. Fortunately, I realized I was missing double quotes surrounding my validation annotations. Lost time: ~30 minutes. 

Fixing the previous example with annotations:

```go
// Maintainer a single maintainer's personal information.
type Maintainer struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
}
```

I also added the `dive` keyword to perform deep validation on the list of maintainers in the `ApplicationMetadata` struct:

```go
type ApplicationMetadata struct {
	Title       string        `validate:"required"`
	Version     string        `validate:"required"`
	Maintainers []*Maintainer `validate:"required,dive,required"`
	Company     string        `validate:"required"`
	Website     string        `validate:"required"`
	Source      string        `validate:"required"`
	License     string        `validate:"required"`
	Description string        `validate:"required"`
}
```

## Validating Input

I can now validate my input and return any issues to the user:

```go
err = validate.Struct(metadata)
if err != nil {
    // If we fail to validate, automatically return 400
    w.WriteHeader(http.StatusBadRequest)
    w.Write([]byte("Failed to validate input of the following parameters:\n"))

    // Be helpful and tell users what fails inside this request
    for _, err := range err.(validator.ValidationErrors) {
        fmt.Fprintf(w, "%s has invalid value %s\n", err.Namespace(), err.Value())
    }
}
```

It seemed appropriate to me to store applications on some unique key, since we are talking about a user describing an application. What would the expected behavior be with two requests to create identical applications? `409 Conflict` or `201 Created`? I don't have a strong personal stance on this and my query implementation works either way.

I chose to use `Title` field as a unique field and handle conflicts by failing to create the application and returning `HTTP 409 Conflict`.

n.b.: delete these lines of code and everything works exactly the same.

```go
// Check if a conflicting application already exists
if util.CheckTitle(applications, metadata.Title) {
    w.WriteHeader(http.StatusConflict)
    fmt.Fprintf(w, "An application with title %s already exists, please use a unique title.", metadata.Title)
    return
}
```

Finally, we can persist this into the global list of applications:

```go
applications = append(applications, *metadata)
```

At this point, server responses are fairly simple.

- 200 (maybe change to 201) for created applications
- 500 for server failures that should not occur
- 400 for bad requests, with some information. 

## Query Endpoint

The search endpoint presents a more interesting challenge. I'd like a clean way to differentiate structs, e.g. using a hash and comparing, but this doesn't seem to save me much since no matter what I need to iterate the fields.

My initial intuition was to handle this as a GET request with query parameters. I considered how I would handle searching for an array of maintainers in this case. I decided handling multiple maintainers would be cleaner using a POST request, parsing out the maintainers, doing an exact match on the other fields and then verifying the maintainers last. This has the added benefit of reusing the same parsing logic for the creation: we simply parse out a struct of the same type, and match the fields to any existing applications to return a list.

### Comparison and Ignoring Null Values

This led me to a desire to cleanly compare structs. I found out this isn't a particularly clean operation in Go, or at least not using reflection. Nonetheless for my first pass I decided to stick with reflection to see where it took me. As currently implemented this solution would fail to cover any sort of cases including nested slices/maps, since it basically does a reflect.DeepEqual over the top level values (ignoring any search values which are null), and only descends a level to compare for maintainers. This means the following edge case is not covered.

Application stored in server: 

```yaml
title: Valid App 1
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
 Some application content, and description
```

Query parameters:

```yaml
title: Valid App 1
version: 0.0.1
maintainers:
- name: firstmaintainer app1 # App without email
company: Random Inc.
website: https://website.com
```

This result would not be returned as a match.

### Improving Checks

As I continued to push my reflection solution I was surprised by the power of Go in ways that I expected would be more restrictive. For example, if the values of two fields aren't equal, we'd like to make sure the user-provided field isn't simply unprovided/set to the zero value for that type. I implemented this as a type switch inside the failure case: 

```go
switch desiredVal.Field(i).Interface().(type) {
case string:
    if desiredVal.Field(i).Interface() != "" {
    return false
    }
case []*types.Maintainer:
    // Do Stuff
default:
    if desiredVal.Field(i).Interface() != nil {
    return false
    }
}
```

I was surprised to find that the middle case actually works. I didn't expect this to be the case, and had planned to special case the `maintainers` field as a workaround. Fortunately, with this in hand all I needed was to implement pairwise comparison to check the user-provided maintainers against the stored application metadata:

```go
case []*types.Maintainer:
    knownMaintainers := knownVal.Field(i).Interface().([]*types.Maintainer)
    foundMaintainers := desiredVal.Field(i).Interface().([]*types.Maintainer)
    for _, foundMaintainer := range foundMaintainers {
        if !Any(knownMaintainers, foundMaintainer, CompareMaintainer) {
        	return false
        }
    }
.
.
.
// Any returns true if the provided maintainer is known to us.
func Any(knowns []*types.Maintainer, desired *types.Maintainer, f func(*types.Maintainer, *types.Maintainer) bool) bool {
	for _, v := range knowns {
		if f(v, desired) {
			return true
		}
	}
	return false
}

// CompareMaintainer returns true if both email and name match a known author, counting comparisons against empty values as true.
func CompareMaintainer(known *types.Maintainer, desired *types.Maintainer) bool {
	knownVal := reflect.ValueOf(known).Elem()
	desiredVal := reflect.ValueOf(desired).Elem()
	fields := knownVal.NumField()

	for i := 0; i < fields; i++ {
		// Unlike Compare for ApplicationMetadata, we should shortcircuit here.
		if !reflect.DeepEqual(knownVal.Field(i).Interface(), desiredVal.Field(i).Interface()) && desiredVal.Field(i).Interface() != "" {
			return false
		}
	}
	return true
}
```

### Response Payload

I opted to simply marshal the matched objects back to YAML and return them. I rarely see APIs use YAML over JSON (or GRPC/protobufs), but it's machine and human-readable so I am fine with this.

```go
matches := util.Filter(applications, metadata, util.Compare)
data, err := yaml.Marshal(matches)
if err != nil {
	http.Error(w, "Failed to marshal search matches. This is likely a server error.", http.StatusInternalServerError)
    return
}
w.WriteHeader(http.StatusOK)
w.Write(data)
return
```

## Testing

### Cases

As I built up my code I manually ran HTTP requests via Postman to verify. My list of basic tests quickly expanded:

- Create with valid metadata returns 201
- Create with valid, conflicting metadata returns 409

- Create with missing field returns 400 and missing field name
- Create with parse error returns 400 and indicates parse error
- Create with fields as invalid types should fail with 400
- Search for exact match returns 200
- Search for single field which matches multiple apps should return them all
- Search parameters where keys all match but one or more values have different values
- Search parameters with wrong types for parameters
- Search for an app with top level field missing should succeed 
- Search for an app with maintainer name or email missing should succeed

### Test Infrastructure/Frameworks

Due to my unfamiliarity with Go, I needed to figure out a way to test my code properly. I found [this article](https://blog.questionable.services/article/testing-http-handlers-go/) covering how one can test handlers using `net/http/httptest` and `testing` packages. I figure if I can learn to test the handlers properly, I'll be well on my way to unit testing vanilla functions as well. 

While scaffolding out some basic tests I found that things would be much easier if I refactored some components to make them more modular. I opted to include a new `Server` type to contain all global state (i.e., persisted application metadata and our validator instance). This enabled me to do setup and tear down of test cases much more easily. For example, I can clear the persisted application metadata on the server between test cases for isolation, or within a single case run multiple requests to test complex scenarios. 

I also found some utility functions greatly simplified the experience. Thanks [@benbjohnson](https://github.com/benbjohnson/testing)!
