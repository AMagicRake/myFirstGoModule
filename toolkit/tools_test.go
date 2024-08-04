package toolkit

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	testTools := &Tools{}

	s := testTools.RandomString(10)

	if len(s) != 10 {
		t.Error("Wrong length random string returned")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{
		name:          "allowed no rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    false,
		errorExpected: false,
	},
	{
		name:          "allowed rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    true,
		errorExpected: false,
	},
	{
		name:          "not allowed",
		allowedTypes:  []string{"image/jpeg"},
		renameFile:    false,
		errorExpected: true,
	},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		//set up a pipe to avoid buggering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer writer.Close()
			defer wg.Done()

			// create the form data field 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		//read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)

		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer writer.Close()

		// create the form data field 'file'
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	//read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFiles, err := testTools.UploadOneFIle(request, "./testdata/uploads/", true)

	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))

}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	path := "test/path/check"
	tool := Tools{}
	err := tool.CreateDirIfNotExists(path)
	if err != nil {
		t.Error(err)
	}
	err = tool.CreateDirIfNotExists(path)
	if err != nil {
		t.Error(err)
	}
	_ = os.Remove((path))
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "Now is the time for all GOOD men! + fish & such &^123", expected: "now-is-the-time-for-all-good-men-fish-such-123", errorExpected: false},
	{name: "japanese string", s: "こんにちは", expected: "", errorExpected: true},
}

func TestTools_Slugify(t *testing.T) {

	tool := Tools{}

	for _, e := range slugTests {
		slug, err := tool.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}

		if !e.errorExpected && slug != e.expected {

			t.Errorf("%s: wrong slug returned expected: %s got:%s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	tool := Tools{}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	tool.DownloadStaticFile(rr, req, "./testdata", "not_a_database.png", "bart_meme.png")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "247441" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"bart_meme.png\"" {
		t.Errorf("wrong content disposition: %s", res.Header["Content-Disposition"][0])
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "good json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted json", json: `{"foo": }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo": 5 }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "bar"}{"foo": "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error in json", json: `{"foo": 1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field", json: `{"fooo": "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field allowed", json: `{"fooo": "1"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 1, allowUnknown: true},
	{name: "not json", json: `Hello World`, errorExpected: true, maxSize: 1024, allowUnknown: false},
}

func TestTools_ReadJSON(t *testing.T) {
	tool := &Tools{}

	for _, e := range jsonTests {

		tool.MaxJsonSize = e.maxSize
		tool.AllowUnknownFields = e.allowUnknown

		decodedJson := struct {
			Foo string `json:"foo"`
		}{}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		rr := httptest.NewRecorder()

		err = tool.ReadJSON(rr, req, &decodedJson)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected but one received: %s", e.name, err.Error())
		}

		req.Body.Close()
	}
}
