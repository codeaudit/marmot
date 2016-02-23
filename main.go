package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// please see the Cloud Vision Documentation to get setup
// https://cloud.google.com/vision/docs/getting-started

//TODO got replace panics with something better

func main() {
	fmt.Println("Initializing marmot checker")
	mux := http.NewServeMux()
	mux.HandleFunc("/post/", postIt)
	http.ListenAndServe(":2332", mux)
}

func postIt(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		fmt.Println("Receving POST request")
		//marshal things
		//log 'em
		urlPathString := r.URL.Path[1:]
		fileName := strings.Split(urlPathString, "/")[1]

		fmt.Printf("Filename to register:\t\t%s\n", fileName)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error reading file body: %v\n", err)
			os.Exit(1)
		}

		imagePNGpath := WriteTempFile(fileName, body)
		defer RemoveTempFile(imagePNGpath)

		ok := CheckIfPNG(imagePNGpath)
		if !ok {
			fmt.Println("image must be of format .png")
			os.Exit(1)
		}

		imgBase64string := ConvertToBase64(imagePNGpath)

		//returns []byte
		payloadJSONbytes := ConstructJSONPayload(imgBase64string)

		env := "CLOUD_VISION_API_KEY"
		apiKey := os.Getenv(env)
		if apiKey == "" {
			fmt.Println("Please set your cloud vision api key as the env var: %s\n", env)
			os.Exit(1)
		}

		url := ConstructURL(apiKey)

		responseFromGoogle := PostToGoogleCloudVisionAPI(url, payloadJSONbytes)

		theImageIs := ParseResponse(responseFromGoogle)

		//[]string
		paramsToCheck := os.Getenv("CLOUD_VISION_MARMOT_CHECKER")

		okay := CheckIfMatched(theImageIs, strings.Split(paramsToCheck, ","))
		if okay {
			//temp file originally uploaded; see above
			PostImageToToadserver(imagePNGpath)
		} else {
			fmt.Printf("The image supplied is a %s which does not match any of %s", theImageIs, strings.Split(paramsToCheck, ","))
			os.Exit(1)
		}

	}
}

// image must be in png format
// adapted from:
// https://www.socketloop.com/tutorials/golang-how-to-verify-uploaded-file-is-image-or-allowed-file-types
func CheckIfPNG(imagePath string) bool {
	var ok bool
	file, err := os.Open(imagePath)
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		os.Exit(1)
	}
	buff := make([]byte, 512) // see see http://golang.org/pkg/net/http/#DetectContentType

	_, err = file.Read(buff)

	filetype := http.DetectContentType(buff)

	if filetype == "image/png" {
		ok = true
	} else {
		ok = false
	}
	return ok
}

// format required for json payload to Cloud Vision
// adapted from:
// https://www.socketloop.com/tutorials/golang-encode-image-to-base64-example
func ConvertToBase64(imagePath string) string {
	imgFile, err := os.Open(imagePath)
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		os.Exit(1)
	}
	defer imgFile.Close()

	// create new buffer based on file size
	//TODO catch err ... ?
	fInfo, _ := imgFile.Stat()
	var size int64 = fInfo.Size()
	buf := make([]byte, size)

	fReader := bufio.NewReader(imgFile)
	fReader.Read(buf)

	// convert the buffer bytes to base64 string
	imgBase64str := base64.StdEncoding.EncodeToString(buf)
	return imgBase64str

}

// conform to the spec in the docs
// https://cloud.google.com/vision/docs/getting-started
// returns []byte for post request
func ConstructJSONPayload(imgBase64string string) []byte {
	jsonPayload := `{"requests":[{"image":{"content":"` + imgBase64string + `"},"features":[{"type":"LABEL_DETECTION","maxResults":1}]}]}`
	return []byte(jsonPayload)
}

//check env var for key & fmt.Sprint
func ConstructURL(apiKey string) string {
	url := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", apiKey)
	return url
}

//return the json response to be parsed later
func PostToGoogleCloudVisionAPI(url string, jsonBytes []byte) string {
	request, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		os.Exit(1)
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("error posting to Google: %v\n", err)
		os.Exit(1)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("error reading body: %v\n", err)
		os.Exit(1)
	}
	return string(body)

}

func ParseResponse(jsonString string) string {
	return ""
	// return "description" field from Google response
}

func CheckIfMatched(theImageIs string, imagesToMatch []string) bool {
	return false
}

func PostImageToToadserver(image string) error {
	return nil
}

// writes temp file for reading as needed
// used in conjunction with defer RemoveTempFile()
func WriteTempFile(fileName string, imageBody []byte) string {
	f, err := ioutil.TempFile("", fileName)
	if err != nil {
		fmt.Printf("error creating temp file: %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(f.Name(), imageBody, 0777); err != nil {
		fmt.Printf("error writing temp file: %v\n", err)
		os.Exit(1)
	}

	return f.Name()
}

// removes the temp file after everything is done
func RemoveTempFile(imagePath string) {
}
