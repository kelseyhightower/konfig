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

	_ "github.com/kelseyhightower/konfig"
)

func F(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "CONFIG_FILE: %s\n", os.Getenv("CONFIG_FILE"))
	fmt.Fprintf(w, "ENVIRONMENT: %s\n", os.Getenv("ENVIRONMENT"))
	fmt.Fprintf(w, "FOO: %s\n\n", os.Getenv("FOO"))

	data, err := ioutil.ReadFile(os.Getenv("CONFIG_FILE"))
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "", 500)
		return
	}

	fmt.Fprintf(w, "# %s\n", os.Getenv("CONFIG_FILE"))
	fmt.Fprintf(w, "%s\n", data)
}
