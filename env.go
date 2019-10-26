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

func getProjectID() string {
	return projectID
}
