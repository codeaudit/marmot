package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// please see the Cloud Vision Documentation to get setup
// https://cloud.google.com/vision/docs/getting-started
// requires that you set CLOUD_VISION_API_KEY as an env var

// server that receives a png image, & sends it to the
// Google Cloud Vision API for processing.
// returns 3 closest match results & checks
// them against a []string of allowed match names set by:
// CLOUD_VISION_MARMOT_CHECKS as env var with comma seperated strings
// if a match is found, file is posted to the toadserver
// which is required to be running on the host.the
// toadserver creates an index of the filename alongside
// the IPFS hash of the file (on the chain) and adds the file to IPFS.

func main() {
	fmt.Println("Initializing marmot checker")
	mux := http.NewServeMux()
	mux.HandleFunc("/postImage/", PostImage)

	port := ":2332"
	fmt.Printf("Listening on port%s\n", port)
	http.ListenAndServe(port, mux)
}

func PostImage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		fmt.Println("Receving POST request")

		urlPathString := r.URL.Path[1:]
		fileName := strings.Split(urlPathString, "/")[1]

		fmt.Printf("Filename:%s\n", fileName)

		// read body of the post (presumably a .png image
		// coming in from `--data-binary` on the post request)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error reading file body: %v\n", err)
			os.Exit(1)
		}
		defer r.Body.Close()

		// needed for some things here, and below;
		// if image passes the checker, this var used
		// as name to register for upload to toadserver
		imagePNGpath := WriteTempFile(fileName, body)
		defer RemoveTempFile(imagePNGpath)

		//exits if image is not .png
		CheckIfPNG(imagePNGpath)

		// used in the json payload, per Google's spec
		imgBase64string := ConvertToBase64(imagePNGpath)

		// returns []byte for post to the cloud vision API
		// contains fields that can be tweaked for image specifications
		payloadJSONbytes := ConstructJSONPayload(imgBase64string)

		// env var for your api key. see the google docs to make one
		// func will exit if env is empty (simple sanity check)
		apiKey := checkEnv("CLOUD_VISION_API_KEY")

		// assembles url from key
		url := ConstructURL(apiKey)

		// send it all away, await a response
		responseFromGoogle := PostToGoogleCloudVisionAPI(url, payloadJSONbytes)

		// []string of descriptors returned by image analysis
		imageDescriptors := ParseResponse(responseFromGoogle)

		// comma seperated string of descriptors you want to match
		toCheck := checkEnv("CLOUD_VISION_MARMOT_CHECKS")

		// split the string to pass in a []string
		// for comparison of two arrays
		formattedOutput, okay := CheckIfMatched(imageDescriptors, strings.Split(toCheck, ","))
		if okay {
			fmt.Println(formattedOutput)
			//temp file originally uploaded; see above
			out := PostImageToToadserver(imagePNGpath)
			fmt.Println(out)
		} else {
			fmt.Println(formattedOutput)
		}
	}
}

// ensure image is in png format
// other format could be supported if desired
func CheckIfPNG(imagePath string) {
	file, err := os.Open(imagePath)
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		os.Exit(1)
	}
	buff := make([]byte, 512) // see see http://golang.org/pkg/net/http/#DetectContentType

	_, err = file.Read(buff)

	filetype := http.DetectContentType(buff)

	if filetype != "image/png" {
		fmt.Println("image must be of format .png")
		os.Exit(1)
	}
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
// returns []byte for upcoming post request
// modify "features" to get a richer response
// see: https://cloud.google.com/vision/docs/concepts
// for details; marshalling of the response (func ParseResponse())
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
func PostToGoogleCloudVisionAPI(url string, jsonBytes []byte) []byte {
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
		fmt.Printf("error reading response.Body: %v\n", err)
		os.Exit(1)
	}

	if response.Status != "200 OK" {
		fmt.Printf("bad status code (not 200), exiting: %s\n", response.Status)
		fmt.Printf("response from Google:\n%s\n", string(body))
		os.Exit(1)
	}
	return body //unmarshalled next
}

type Labels struct {
	Responses []interface{} `json:"responses"`
}

// Unmarshal response from Google
// hacky but works
// could break if API changes
func ParseResponse(jsonBytes []byte) []string {

	var labels Labels
	if err := json.Unmarshal(jsonBytes, &labels); err != nil {
		fmt.Printf("error unmarshalling response: %v\n", err)
		os.Exit(1)
	}

	out := labels.Responses[0].(map[string]interface{})
	epic := out["labelAnnotations"].([]interface{})

	descriptions := make([]string, len(epic))
	for i, maP := range epic {
		theMap := maP.(map[string]interface{})
		thingIactuallyWant := theMap["description"]
		descriptions[i] = thingIactuallyWant.(string)
		//XXX somehow a duplicate fourth slot sneaks in ... ?
	}
	return descriptions
}

// compares images descriptors returned from google
// to the multi-word string given as the following env var:
// CLOUD_VISION_MARMOT_CHECKS
// with comma seperated words
func CheckIfMatched(imageDescriptors, toCheckAgainst []string) (string, bool) {
	var output string
	var ok bool

	toCheck := make(map[string]bool)
	for _, tca := range toCheckAgainst {
		toCheck[tca] = true
	}

	for _, id := range imageDescriptors {
		if toCheck[id] == true {
			ok = true
			break
		} else {
			ok = false
		}
	}

	forBoth := fmt.Sprintf("Descriptors: %v\nTo Check: %v\n", imageDescriptors, toCheckAgainst)

	if ok == true {
		output = `Success! 
The image has descriptors that matched with the supplied check parameters.
` + forBoth + `
Gonna post to toadserver...`
	} else {
		output = `Sad marmot :( 
The image has descriptors that did not match with the supplied check parameters.
` + forBoth + `
Not posting to toadserver. Try again with a new image, or different check parameters.`
	}
	return output, ok
}

// requires a running toadserver: `eris services start toadserver`
// and additional chain configuration: `eris services cat toadserver`
// for more information. alternatively, you can `bash run.sh` the
// script in this repository to configure and initialize
// all the required dependencies.
func PostImageToToadserver(imagePNGpath string) string {
	imageBytes, err := ioutil.ReadFile(imagePNGpath)
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		os.Exit(1)
	}

	formatName := strings.Split(imagePNGpath, "/") // format because temp file
	fnName := formatName[len(formatName)-1]
	tsIP := checkEnv("TOADSERVER_HOST")
	url := fmt.Sprintf("http://%s:11113/postfile/%s", tsIP, fnName)
	fmt.Printf("Posting to toadserver at url: %s\n", url)

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(imageBytes))
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{}

	_, err = client.Do(request) // response.Body will be  empty
	if err != nil {
		fmt.Printf("error posting to toadserver: %v\n", err)
		os.Exit(1)
	}

	return "Success posting to toadserver!"
}

// writes temp file for reading as needed
// used in conjunction with defer RemoveTempFile() below
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

// exit if env is empty
// return the value from env
func checkEnv(env string) (envVar string) {
	envVar = os.Getenv(env)
	if envVar == "" {
		fmt.Println("Please read the documentation and set this env var: %s\n", env)
		os.Exit(1)
	}
	return envVar
}
