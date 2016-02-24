package main

import (
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

func main() {
	fmt.Println("Initializing marmot checker")
	mux := http.NewServeMux()
	mux.HandleFunc("/postImage/", PostImage)

	port := "2332"
	p := fmt.Sprintf(":%s", port)
	fmt.Printf("Listening on port: %s\n", port)
	http.ListenAndServe(p, mux)
}

// receives a png image, & sends it to the Google Cloud Vision API
// for processing. returns closest match result & checks
// it against a []string of allowed match names.
// if a match is found, file is posted to the toadserver
// which is required to be running on the host
// the toadserver creates an index of the filename alongside
// the IPFS hash of the file and adds the file to IPFS
func PostImage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		fmt.Println("Receving POST request")
		//get the
		urlPathString := r.URL.Path[1:]
		fileName := strings.Split(urlPathString, "/")[1]

		fmt.Printf("Filename:\t\t%s\n", fileName)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error reading file body: %v\n", err)
			os.Exit(1)
		}
		defer r.Body.Close()

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

		fmt.Println(responseFromGoogle)

		//theImageIs := ParseResponse(responseFromGoogle)

		/*
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
		*/
	}
}

// image must be in png format
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
func ConvertToBase64(imagePath string) string {
	imageBytes, err := ioutil.ReadFile(imagePath)
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		os.Exit(1)
	}

	// convert the buffer bytes to base64 string
	imgBase64str := base64.StdEncoding.EncodeToString(imageBytes)
	return imgBase64str
}

// conforms to the spec in the docs
// https://cloud.google.com/vision/docs/getting-started
// returns []byte for post request
// modify "features" to get a richer response
// see: https://cloud.google.com/vision/docs/concepts
// for details; marshalling of the response
// would need to be modified accordingly
func ConstructJSONPayload(imgBase64string string) []byte {
	jsonPayload := `{"requests":[{"image":{"content":"` + imgBase64string + `"},"features":[{"type":"LABEL_DETECTION","maxResults":3}]}]}`
	return []byte(jsonPayload)
}

// format URL with apiKey
func ConstructURL(apiKey string) string {
	return fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", apiKey)
}

// POST with URL from above and jsonPayload
// returns the whole json response to be parsed next
func PostToGoogleCloudVisionAPI(url string, jsonBytes []byte) string {
	fmt.Println("Posting to google cloud vision API")
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
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

// json.Unmarshal
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
	if err := os.Remove(imagePath); err != nil {
		fmt.Printf("error removing file: %v\n", err)
		os.Exit(1)
	}
}
