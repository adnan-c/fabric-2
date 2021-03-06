/*
Copyright IBM Corp. 2016, 2017 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package couchdb

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"unicode/utf8"

	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	ledgertestutil "github.com/hyperledger/fabric/core/ledger/testutil"
)

//Basic setup to test couch
var connectURL = "localhost:5984"
var badConnectURL = "localhost:5990"
var database = "couch_util_testdb"
var username = ""
var password = ""

func cleanup() {
	//create a new connection
	couchInstance, _ := CreateCouchInstance(connectURL, username, password)
	db, _ := CreateCouchDatabase(*couchInstance, database)
	//drop the test database
	db.DropDatabase()
}

type Asset struct {
	ID        string `json:"_id"`
	Rev       string `json:"_rev"`
	AssetName string `json:"asset_name"`
	Color     string `json:"color"`
	Size      string `json:"size"`
	Owner     string `json:"owner"`
}

var assetJSON = []byte(`{"asset_name":"marble1","color":"blue","size":"35","owner":"jerry"}`)

func TestMain(m *testing.M) {
	ledgertestutil.SetupCoreYAMLConfig("./../../../../peer")
	statedb := os.Getenv("CORE_LEDGER_STATE_STATEDATABASE")
	if statedb == "CouchDB" {
		os.Setenv("core.ledger.state.statedatabase","CouchDB" )
		connectURL = os.Getenv("CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS")
		username = os.Getenv("CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME")
		password = os.Getenv("CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD")
	}
	os.Exit(m.Run())
}

func TestDBConnectionDef(t *testing.T) {

	//call a helper method to load the core.yaml
	ledgertestutil.SetupCoreYAMLConfig("./../../../../peer")

	//create a new connection
	_, err := CreateConnectionDefinition(connectURL, "", "")
	testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create database connection definition"))

}

func TestDBBadConnectionDef(t *testing.T) {

	//create a new connection
	_, err := CreateConnectionDefinition("^^^localhost:5984", "", "")
	testutil.AssertError(t, err, fmt.Sprintf("Did not receive error when trying to create database connection definition with a bad hostname"))

}

func TestDBCreateSaveWithoutRevision(t *testing.T) {

	if ledgerconfig.IsCouchDBEnabled() == true {

		cleanup()
		defer cleanup()

		//create a new instance and database object
		couchInstance, err := CreateCouchInstance(connectURL, username, password)
		testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create couch instance"))
		db := CouchDatabase{couchInstance: *couchInstance, dbName: database}

		//create a new database
		_, errdb := db.CreateDatabaseIfNotExist()
		testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to create database"))

		//Save the test document
		_, saveerr := db.SaveDoc("2", "", assetJSON, nil)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save a document"))

	}
}

func TestDBBadConnection(t *testing.T) {

	if ledgerconfig.IsCouchDBEnabled() == true {

		//create a new instance and database object
		couchInstance, err := CreateCouchInstance(badConnectURL, username, password)
		testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create couch instance"))
		db := CouchDatabase{couchInstance: *couchInstance, dbName: database}

		//create a new database
		_, errdb := db.CreateDatabaseIfNotExist()
		testutil.AssertError(t, errdb, fmt.Sprintf("Error should have been thrown while creating a database with an invalid connecion"))

		//Save the test document
		_, saveerr := db.SaveDoc("3", "", assetJSON, nil)
		testutil.AssertError(t, saveerr, fmt.Sprintf("Error should have been thrown while saving a document with an invalid connecion"))

		//Retrieve the updated test document
		_, _, geterr := db.ReadDoc("3")
		testutil.AssertError(t, geterr, fmt.Sprintf("Error should have been thrown while retrieving a document with an invalid connecion"))

	}
}

func TestDBCreateDatabaseAndPersist(t *testing.T) {

	if ledgerconfig.IsCouchDBEnabled() == true {

		cleanup()
		defer cleanup()

		//create a new instance and database object
		couchInstance, err := CreateCouchInstance(connectURL, username, password)
		testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create couch instance"))
		db := CouchDatabase{couchInstance: *couchInstance, dbName: database}

		//create a new database
		_, errdb := db.CreateDatabaseIfNotExist()
		testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to create database"))

		//Retrieve the info for the new database and make sure the name matches
		dbResp, _, errdb := db.GetDatabaseInfo()
		testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to retrieve database information"))
		testutil.AssertEquals(t, dbResp.DbName, database)

		//Save the test document
		_, saveerr := db.SaveDoc("idWith/slash", "", assetJSON, nil)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save a document"))

		//Retrieve the test document
		dbGetResp, _, geterr := db.ReadDoc("idWith/slash")
		testutil.AssertNoError(t, geterr, fmt.Sprintf("Error when trying to retrieve a document"))

		//Unmarshal the document to Asset structure
		assetResp := &Asset{}
		json.Unmarshal(dbGetResp, &assetResp)

		//Verify the owner retrieved matches
		testutil.AssertEquals(t, assetResp.Owner, "jerry")

		//Save the test document
		_, saveerr = db.SaveDoc("1", "", assetJSON, nil)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save a document"))

		//Retrieve the test document
		dbGetResp, _, geterr = db.ReadDoc("1")
		testutil.AssertNoError(t, geterr, fmt.Sprintf("Error when trying to retrieve a document"))

		//Unmarshal the document to Asset structure
		assetResp = &Asset{}
		json.Unmarshal(dbGetResp, &assetResp)

		//Verify the owner retrieved matches
		testutil.AssertEquals(t, assetResp.Owner, "jerry")

		//Change owner to bob
		assetResp.Owner = "bob"

		//create a byte array of the JSON
		assetDocUpdated, _ := json.Marshal(assetResp)

		//Save the updated test document
		_, saveerr = db.SaveDoc("1", "", assetDocUpdated, nil)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save the updated document"))

		//Retrieve the updated test document
		dbGetResp, _, geterr = db.ReadDoc("1")
		testutil.AssertNoError(t, geterr, fmt.Sprintf("Error when trying to retrieve a document"))

		//Unmarshal the document to Asset structure
		assetResp = &Asset{}
		json.Unmarshal(dbGetResp, &assetResp)

		//Assert that the update was saved and retrieved
		testutil.AssertEquals(t, assetResp.Owner, "bob")

		//Drop the database
		_, errdbdrop := db.DropDatabase()
		testutil.AssertNoError(t, errdbdrop, fmt.Sprintf("Error dropping database"))

		//Retrieve the info for the new database and make sure the name matches
		_, _, errdbinfo := db.GetDatabaseInfo()
		testutil.AssertError(t, errdbinfo, fmt.Sprintf("Error should have been thrown for missing database"))

	}

}

func TestDBBadJSON(t *testing.T) {

	if ledgerconfig.IsCouchDBEnabled() == true {

		cleanup()
		defer cleanup()

		//create a new instance and database object
		couchInstance, err := CreateCouchInstance(connectURL, username, password)
		testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create couch instance"))
		db := CouchDatabase{couchInstance: *couchInstance, dbName: database}

		//create a new database
		_, errdb := db.CreateDatabaseIfNotExist()
		testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to create database"))

		//Retrieve the info for the new database and make sure the name matches
		dbResp, _, errdb := db.GetDatabaseInfo()
		testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to retrieve database information"))
		testutil.AssertEquals(t, dbResp.DbName, database)

		badJSON := []byte(`{"asset_name"}`)

		//Save the test document
		_, saveerr := db.SaveDoc("1", "", badJSON, nil)
		testutil.AssertError(t, saveerr, fmt.Sprintf("Error should have been thrown for a bad JSON"))

	}

}

func TestPrefixScan(t *testing.T) {
	if !ledgerconfig.IsCouchDBEnabled() {
		return
	}
	cleanup()
	defer cleanup()

	//create a new instance and database object
	couchInstance, err := CreateCouchInstance(connectURL, username, password)
	testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create couch instance"))
	db := CouchDatabase{couchInstance: *couchInstance, dbName: database}

	//create a new database
	_, errdb := db.CreateDatabaseIfNotExist()
	testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to create database"))

	//Retrieve the info for the new database and make sure the name matches
	dbResp, _, errdb := db.GetDatabaseInfo()
	testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to retrieve database information"))
	testutil.AssertEquals(t, dbResp.DbName, database)

	//Save documents
	for i := 0; i < 20; i++ {
		id1 := string(0) + string(i) + string(0)
		id2 := string(0) + string(i) + string(1)
		id3 := string(0) + string(i) + string(utf8.MaxRune-1)
		_, saveerr := db.SaveDoc(id1, "", assetJSON, nil)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save a document"))
		_, saveerr = db.SaveDoc(id2, "", assetJSON, nil)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save a document"))
		_, saveerr = db.SaveDoc(id3, "", assetJSON, nil)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save a document"))

	}
	startKey := string(0) + string(10)
	endKey := startKey + string(utf8.MaxRune)
	resultsPtr, geterr := db.ReadDocRange(startKey, endKey, 1000, 0)
	testutil.AssertNoError(t, geterr, fmt.Sprintf("Error when trying to perform a range scan"))
	testutil.AssertNotNil(t, resultsPtr)
	results := *resultsPtr
	testutil.AssertEquals(t, len(results), 3)
	testutil.AssertEquals(t, results[0].ID, string(0)+string(10)+string(0))
	testutil.AssertEquals(t, results[1].ID, string(0)+string(10)+string(1))
	testutil.AssertEquals(t, results[2].ID, string(0)+string(10)+string(utf8.MaxRune-1))

	//Drop the database
	_, errdbdrop := db.DropDatabase()
	testutil.AssertNoError(t, errdbdrop, fmt.Sprintf("Error dropping database"))

	//Retrieve the info for the new database and make sure the name matches
	_, _, errdbinfo := db.GetDatabaseInfo()
	testutil.AssertError(t, errdbinfo, fmt.Sprintf("Error should have been thrown for missing database"))
}

func TestDBSaveAttachment(t *testing.T) {

	if ledgerconfig.IsCouchDBEnabled() == true {

		cleanup()
		defer cleanup()
		byteText := []byte(`This is a test document.  This is only a test`)

		attachment := Attachment{}
		attachment.AttachmentBytes = byteText
		attachment.ContentType = "text/plain"
		attachment.Name = "valueBytes"

		attachments := []Attachment{}
		attachments = append(attachments, attachment)

		//create a new instance and database object
		couchInstance, err := CreateCouchInstance(connectURL, username, password)
		testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create couch instance"))
		db := CouchDatabase{couchInstance: *couchInstance, dbName: database}

		//create a new database
		_, errdb := db.CreateDatabaseIfNotExist()
		testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to create database"))

		//Save the test document
		_, saveerr := db.SaveDoc("10", "", nil, attachments)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save a document"))

		//Attempt to retrieve the updated test document with attachments
		returnDoc, _, geterr2 := db.ReadDoc("10")
		testutil.AssertNoError(t, geterr2, fmt.Sprintf("Error when trying to retrieve a document with attachment"))

		//Test to see that the result from CouchDB matches the initial text
		testutil.AssertEquals(t, string(returnDoc), string(byteText))

	}
}

func TestDBDeleteDocument(t *testing.T) {

	if ledgerconfig.IsCouchDBEnabled() == true {

		cleanup()
		defer cleanup()

		//create a new instance and database object
		couchInstance, err := CreateCouchInstance(connectURL, username, password)
		testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create couch instance"))
		db := CouchDatabase{couchInstance: *couchInstance, dbName: database}

		//create a new database
		_, errdb := db.CreateDatabaseIfNotExist()
		testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to create database"))

		//Save the test document
		_, saveerr := db.SaveDoc("2", "", assetJSON, nil)
		testutil.AssertNoError(t, saveerr, fmt.Sprintf("Error when trying to save a document"))

		//Attempt to retrieve the test document
		_, _, readErr := db.ReadDoc("2")
		testutil.AssertNoError(t, readErr, fmt.Sprintf("Error when trying to retrieve a document with attachment"))

		//Delete the test document
		deleteErr := db.DeleteDoc("2", "")
		testutil.AssertNoError(t, deleteErr, fmt.Sprintf("Error when trying to delete a document"))

		//Attempt to retrieve the test document
		readValue, _, _ := db.ReadDoc("2")
		testutil.AssertNil(t, readValue)

	}
}

func TestDBDeleteNonExistingDocument(t *testing.T) {

	if ledgerconfig.IsCouchDBEnabled() == true {

		cleanup()
		defer cleanup()

		//create a new instance and database object
		couchInstance, err := CreateCouchInstance(connectURL, username, password)
		testutil.AssertNoError(t, err, fmt.Sprintf("Error when trying to create couch instance"))
		db := CouchDatabase{couchInstance: *couchInstance, dbName: database}

		//create a new database
		_, errdb := db.CreateDatabaseIfNotExist()
		testutil.AssertNoError(t, errdb, fmt.Sprintf("Error when trying to create database"))

		//Save the test document
		deleteErr := db.DeleteDoc("2", "")
		testutil.AssertNoError(t, deleteErr, fmt.Sprintf("Error when trying to delete a non existing document"))
	}
}
