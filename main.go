package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("Initializing marmot checker")
	mux := http.NewServeMux()
	mux.HandleFunc("/post/", postIt)
	http.ListenAndServe(":2332", mux)
}

func postIt(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		fmt.Println("Receving POST request")

		ConvertToBase64(image)

	}

}
