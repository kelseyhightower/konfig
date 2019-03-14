// Copyright 2019 The Konfig Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package function

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/kelseyhightower/konfig"
)

func F(w http.ResponseWriter, r *http.Request) {
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		w.Header().Set(pair[0], pair[1])
	}

	data, err := ioutil.ReadFile(os.Getenv("CONFIG_FILE"))
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "", 500)
		return
	}

	fmt.Fprintf(w, "  %s\n", data)
}
