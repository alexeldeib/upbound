package util

import (
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/alexeldeib/upbound/pkg/types"
)

// CheckTitle returns true if the title is in use by an existing application.
func CheckTitle(vs []*types.ApplicationMetadata, title string) bool {
	for _, v := range vs {
		if v.Title == title {
			return true
		}
	}
	return false
}

// Compare checks equality between an existing application and a search query, ignoring null values in the desired query.
func Compare(known *types.ApplicationMetadata, desired *types.ApplicationMetadata) bool {
	// Painful, unsure of a better way to execute this.
	// On reflection (no pun intended), could statically declare an array of key values to check.
	knownVal := reflect.ValueOf(known).Elem()
	desiredVal := reflect.ValueOf(desired).Elem()
	numFields := knownVal.NumField()

	// Iterate the values of the reflected fields, ignoring null but failing immediately on unequal fields.
	for i := 0; i < numFields; i++ {
		// Useful debug for edge cases, extraneous for any real use case
		log.WithFields(log.Fields{"knownField": knownVal.Field(i).Interface()}).Debug("Known")
		log.WithFields(log.Fields{"desiredField": desiredVal.Field(i).Interface()}).Debug("Desired")
		log.WithFields(log.Fields{"equality": !reflect.DeepEqual(knownVal.Field(i).Interface(), desiredVal.Field(i).Interface()), "nullity": !reflect.DeepEqual(desiredVal.Field(i).Interface(), reflect.Zero(desiredVal.Type().Field(i).Type))}).Debug("Result of attempt")
		// test, err := desiredVal.Type().Field(i)
		// log.WithFields(log.Fields{"desiredType": test.Type.Name, "err": err}).Debug("desiredType")
		log.WithFields(log.Fields{"zeroVal": desiredVal.Type().Field(i).Type}).Debug("Zero Val")

		// We want to check equality BUT ignore the field if it wasn't in the user input.
		if !reflect.DeepEqual(knownVal.Field(i).Interface(), desiredVal.Field(i).Interface()) {
			switch desiredVal.Field(i).Interface().(type) {
			case string:
				if desiredVal.Field(i).Interface() != "" {
					return false
				}
			default:
				if desiredVal.Field(i).Interface() != nil {
					return false
				}
			}
		}
	}
	return true
}

// Filter removes elements which are unequal after ignoring null values.
func Filter(knowns []*types.ApplicationMetadata, desired *types.ApplicationMetadata, f func(*types.ApplicationMetadata, *types.ApplicationMetadata) bool) []*types.ApplicationMetadata {
	filtered := make([]*types.ApplicationMetadata, 0)
	for _, v := range knowns {
		if f(v, desired) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

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
		log.WithFields(log.Fields{"knownField": knownVal.Field(i).Interface()}).Debug("Known")
		log.WithFields(log.Fields{"desiredField": desiredVal.Field(i).Interface()}).Debug("Desired")
		log.WithFields(log.Fields{"equality": reflect.DeepEqual(knownVal.Field(i).Interface(), desiredVal.Field(i).Interface()), "nullity": desiredVal.Field(i).Interface() != nil}).Debug("Result of attempt")

		// Unlike Compare for ApplicationMetadata, we should shortcircuit here.
		if !reflect.DeepEqual(knownVal.Field(i).Interface(), desiredVal.Field(i).Interface()) && desiredVal.Field(i).Interface() != nil {
			return false
		}
	}
	return true
}
