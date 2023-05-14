package main

import (
	"encoding/json"
	"fmt"
	"log"
)

func Pprint(obj interface{}) {
	json, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("%s\n", string(json))
}

// func HandleError(err error, )
