# Upbound take home log

## Initial Impression

Breaking down the assignment, it can map fairly closely onto a few component pieces.

- Server, probably with a PUT/POST endpoint for creating application metadata and a GET endpoint for searching it using query parameters. Seems safe to stick with `net/http`.
- Parsing of YAML input. I will defer this to an external dependency, probably [go-yaml](https://github.com/go-yaml/yaml).
- Input validation. Given the validation appears fairly simple on first glance, I could manage this without any additional dependencies. Otherwise I will likely defer this to something like [go-validator](https://github.com/go-playground/validator).
- Simple logging for my own debugging and sanity. The usual `log` or `glog` would be plenty here. I may instead use the opportunity to familiarize myself with [logrus](https://github.com/sirupsen/logrus)
- Struct(s) to define application metadata shape.
- An implementation of filtering/searching based on query parameters for the get endpoint. Not sure on first pass how I would like to achieve this. 

## First steps

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

