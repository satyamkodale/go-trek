package main

import (
	"fmt"
	"log"

	mgo "gopkg.in/mgo.v2"
)

var db *mgo.Database

const (
	hostName       string = "localhost:2107"
	dbName         string = "TrekDB"
	collectionName string = "trek"
	port           string = ":9000"
)

func init() {
	sess, err := mgo.Dial(hostName)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
	db = sess.DB(dbName)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	fmt.Println("Setup done")
}
