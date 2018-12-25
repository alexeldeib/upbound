package types

// Maintainer a single maintainer's personal information.
type Maintainer struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
}

// ApplicationMetadata describes the required information to provision an application.
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
