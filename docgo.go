// Package docgo uses tls/http requests to connect and manipulate document db instances.
package docgo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"
)

// Database holds individual Database responses from docdb
type Database struct {
	ID string `json:"id"`
}

// DBListResponse holds the responses for the ListDatabases method
type DBListResponse struct {
	Databases []Database `json:"Databases"`
}

// Session is a publicly available struct
type Session struct {
	// parameters go here in form:
	// [name] [type], e.g: foo string, Uppercase for publicly available and lowercase for private.
	Client *http.Client
	Key    string
	URI    string
}

// New initializes a new Session object, it demonstrates how new objects are
// created in Golang
func New(connString string) (Session, error) {
	client := &http.Client{}
	sessionParams := strings.Split(connString, "AccountKey=")

	uri := strings.Trim(strings.TrimPrefix(sessionParams[0], "AccountEndpoint="), "/;")
	key := strings.Trim(sessionParams[1], ";")
	fmt.Println(key)
	s := Session{Client: client, URI: uri, Key: key}

	return s, nil
}

// GetWithHeaders constructs a http client and sends a request with the passed
// in parameters for the header
func (s Session) GetWithHeaders(headerParams map[string]string, url string) (*http.Response, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for i, x := range headerParams {
		request.Header.Add(i, x)
	}
	resp, err := s.Client.Do(request)
	return resp, err
}

//GenerateAuthToken git git git grrrah
func (s Session) GenerateAuthToken(verb, resourceID, resourceType string) (string, error) {
	timeNow := time.Now().UTC().Format(time.RFC1123)
	timeUsed := timeNow[:len(timeNow)-3] + "GMT"
	x := strings.ToLower(verb) + "\n" + strings.ToLower(resourceType) + "\n" + resourceID + "\n" + strings.ToLower(timeUsed) + "\n" + "" + "\n"
	fmt.Println(x)
	var keyUsed []byte
	keyUsed, err := base64.StdEncoding.DecodeString(s.Key)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, keyUsed)
	var buff []byte
	mac.Write([]byte(x))
	buff = mac.Sum(nil)
	signature := base64.StdEncoding.EncodeToString(buff)
	fmt.Println(signature)
	masterToken := "master"
	tokenVersion := "1.0"
	uri := ("type=" + masterToken + "&ver=" + tokenVersion + "&sig=" + signature)
	return uri, nil
}

// ListDatabases lists the databases for the URI/Key combo in the session instance.
func (s Session) ListDatabases() (*DBListResponse, error) {
	x, err := s.GenerateAuthToken("GET", "", "dbs")
	if err != nil {
		return nil, err
	}
	timeNow := time.Now().UTC().Format(time.RFC1123)
	timeUsed := timeNow[:len(timeNow)-3] + "GMT"
	headers := make(map[string]string)
	headers["Authorization"] = x
	headers["x-ms-version"] = "2015-12-16"
	headers["x-ms-date"] = timeUsed
	resp, err := s.GetWithHeaders(headers, s.URI+"/dbs")
	if err != nil {
		return nil, err
	}
	var out DBListResponse
	if resp.StatusCode < 201 {
		err = jsonUnmarshal(resp.Body, &out)
	}
	if resp.StatusCode >= 400 {
		errResp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		return nil, errors.New("Request failed, json returned was: " + string(errResp))
	}
	return &out, nil
}

// GetDatabase Grabs the Database object mapping to the id passed in and returns a Database object.
func (s Session) GetDatabase(id string) (*Database, error) {
	resourceID := path.Join("dbs", id)
	x, err := s.GenerateAuthToken("GET", resourceID, "dbs")
	if err != nil {
		return nil, err
	}
	timeNow := time.Now().UTC().Format(time.RFC1123)
	timeUsed := timeNow[:len(timeNow)-3] + "GMT"
	headers := make(map[string]string)
	headers["Authorization"] = x
	headers["x-ms-version"] = "2015-12-16"
	headers["x-ms-date"] = timeUsed
	resp, err := s.GetWithHeaders(headers, s.URI+"/dbs/"+id)
	if err != nil {
		return nil, err
	}
	var out Database
	if resp.StatusCode < 201 {
		err = jsonUnmarshal(resp.Body, &out)
	}
	if resp.StatusCode >= 400 {
		errResp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		return nil, errors.New("Request failed, json returned was: " + string(errResp))
	}
	return &out, nil
}

func jsonUnmarshal(body io.Reader, v interface{}) error {
	data, _ := ioutil.ReadAll(body)
	fmt.Println(string(data))
	return json.Unmarshal(data, v)
}
