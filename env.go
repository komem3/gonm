/*
 * Copyright (c) 2019 The Gonm Author
 *
 * File for managing project IDs
 */

package gonm

import "os"

var projectID string

func init() {
	projectID = os.Getenv("DATASTORE_PROJECT_ID")
}

// SetProjectID set datastore project id.
//
// It is not necessary to use if the project ID is set in the environment variable DATASTORE_PROJECT_ID.
func SetProjectID(ID string) {
	projectID = ID
}

func getProjectID() string {
	return projectID
}
