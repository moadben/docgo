// Package docgo uses tls/http requests to connect and manipulate document db instances.
package docgo

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Database holds individual Database responses from docdb
type Database struct {
	ID     string `json:"id"`
	Key    string
	Client *http.Client
	URI    string
}

// Collection holds individual Collection responses from docdb
type Collection struct {
	ID     string `json:"id"`
	Key    string
	Client *http.Client
	URI    string
}

// DBListResponse holds the responses for the ListDatabases method
type DBListResponse struct {
	Databases []Database `json:"Databases"`
}

// CollListResponse holds the responses for the ListCollections method
type CollListResponse struct {
	Databases []Collection `json:"DocumentCollections"`
}

// Session is a publicly available struct
type Session struct {
	// parameters go here in form:
	// [name] [type], e.g: foo string, Uppercase for publicly available and lowercase for private.
	Client *http.Client
	Key    string
	URI    string
}

// New creates a new session object out of a DocumentDB connection string
func New(connString string) (Session, error) {
	client := &http.Client{}
	sessionParams := strings.Split(connString, "AccountKey=")

	uri := strings.Trim(strings.TrimPrefix(sessionParams[0], "AccountEndpoint="), "/;")
	key := strings.Trim(sessionParams[1], ";")
	s := Session{Client: client, URI: uri, Key: key}

	return s, nil
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

// ReqWithHeaders constructs a http client and sends a request with the passed
// in parameters for the header
func ReqWithHeaders(headerParams map[string]string, verb, url string, client *http.Client) (*http.Response, error) {
	request, err := http.NewRequest(verb, url, nil)
	if err != nil {
		return nil, err
	}
	for i, x := range headerParams {
		request.Header.Add(i, x)
	}
	resp, err := client.Do(request)
	return resp, err
}

// PostWithHeaders constructs a http client and sends a request with the passed
// in parameters for the header
func PostWithHeaders(headerParams map[string]string, verb, url string, client *http.Client, body io.ReadCloser) (*http.Response, error) {
	request, err := http.NewRequest(verb, url, body)
	if err != nil {
		return nil, err
	}
	for i, x := range headerParams {
		request.Header.Add(i, x)
	}
	resp, err := client.Do(request)
	return resp, err
}

//GenerateAuthToken git git git grrrah
func GenerateAuthToken(verb, resourceID, resourceType, key string) (string, string, error) {
	timeNow := time.Now().UTC().Format(time.RFC1123)
	timeUsed := timeNow[:len(timeNow)-3] + "GMT"

	x := fmt.Sprintf("%s\n%s\n%s\n%s\n\n",
		strings.ToLower(verb),
		strings.ToLower(resourceType),
		resourceID,
		strings.ToLower(timeUsed))

	fmt.Println(x)
	var keyUsed []byte
	keyUsed, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", "", err
	}
	mac := hmac.New(sha256.New, keyUsed)
	var buff []byte
	mac.Write([]byte(x))
	buff = mac.Sum(nil)
	signature := base64.StdEncoding.EncodeToString(buff)
	masterToken := "master"
	tokenVersion := "1.0"
	uri := url.QueryEscape("type=" + masterToken + "&ver=" + tokenVersion + "&sig=" + signature)
	return uri, timeUsed, nil
}

// ListDatabases lists the databases for the URI/Key combo in the session instance.
func (s Session) ListDatabases() (*DBListResponse, error) {
	x, timeUsed, err := GenerateAuthToken("GET", "", "dbs", s.Key)
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	headers["Authorization"] = x
	headers["x-ms-version"] = "2015-12-16"
	headers["x-ms-date"] = timeUsed
	resp, err := ReqWithHeaders(headers, "GET", s.URI+"/dbs", s.Client)
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
		return nil, errors.New("Request to list databases failed, json returned was: " + string(errResp))
	}
	return &out, nil
}

// GetDatabase Grabs the Database object mapping to the id passed in and returns a Database object.
func (s Session) GetDatabase(id string) (*Database, error) {
	resourceID := path.Join("dbs", id)
	x, timeUsed, err := GenerateAuthToken("GET", resourceID, "dbs", s.Key)
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	headers["Authorization"] = x
	headers["x-ms-version"] = "2015-12-16"
	headers["x-ms-date"] = timeUsed
	resp, err := ReqWithHeaders(headers, "GET", s.URI+"/dbs/"+id, s.Client)
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
		return nil, errors.New("Request to get database " + id + " failed, json returned was: " + string(errResp))
	}
	out.Key = s.Key
	out.URI = s.URI
	out.Client = s.Client
	return &out, nil
}

// ListCollections lists all the collections belonging to the Database object
func (d Database) ListCollections() (*CollListResponse, error) {
	resourceID := path.Join("dbs", d.ID)
	x, timeUsed, err := GenerateAuthToken("GET", resourceID, "colls", d.Key)
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	headers["Authorization"] = x
	headers["x-ms-version"] = "2015-12-16"
	headers["x-ms-date"] = timeUsed
	resp, err := ReqWithHeaders(headers, "GET", d.URI+"/dbs/"+d.ID+"/colls", d.Client)
	if err != nil {
		return nil, err
	}
	var out CollListResponse
	if resp.StatusCode < 201 {
		err = jsonUnmarshal(resp.Body, &out)
	}
	if resp.StatusCode >= 400 {
		errResp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		return nil, errors.New("Request to list collections failed, json returned was: " + string(errResp))
	}
	return &out, nil
}

// GetCollection Grabs the Collection object mapping to the id passed in and returns a Collection object.
func (d Database) GetCollection(id string) (*Collection, error) {
	resourceID := path.Join("dbs", d.ID, "colls", id)
	x, timeUsed, err := GenerateAuthToken("GET", resourceID, "colls", d.Key)
	if err != nil {
		return nil, err
	}
	collID := []byte(id)
	var body io.ReadCloser
	body.Read(collID)
	headers := make(map[string]string)
	headers["Authorization"] = x
	headers["x-ms-version"] = "2015-12-16"
	headers["x-ms-date"] = timeUsed
	resp, err := ReqWithHeaders(headers, "GET", d.URI+"/dbs/"+d.ID+"/colls/"+id, d.Client)
	if err != nil {
		return nil, err
	}
	var out Collection
	if resp.StatusCode < 201 {
		err = jsonUnmarshal(resp.Body, &out)
	}
	if resp.StatusCode >= 400 {
		errResp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		return nil, errors.New("Request to get collection " + id + " failed, json returned was: " + string(errResp))
	}
	out.Key = d.Key
	out.URI = d.URI
	out.Client = d.Client
	return &out, nil
}

//CreateCollection creates a collection DUH
func (d Database) CreateCollection(id string) (*Collection, error) {
	resourceID := path.Join("dbs", d.ID)
	x, timeUsed, err := GenerateAuthToken("POST", resourceID, "colls", d.Key)
	if err != nil {
		return nil, err
	}
	reqID := struct {
		ID string `json:"id"`
	}{
		id,
	}
	buff, err := json.Marshal(reqID)
	if err != nil {
		return nil, err
	}
	body := nopCloser{bytes.NewBuffer(buff)}
	headers := make(map[string]string)
	headers["Authorization"] = x
	headers["x-ms-version"] = "2015-12-16"
	headers["x-ms-date"] = timeUsed
	headers["Content-Type"] = "application/query+json"
	resp, err := PostWithHeaders(headers, "POST", d.URI+"/dbs/"+d.ID+"/colls", d.Client, body)
	if err != nil {
		return nil, err
	}
	var out Collection
	if resp.StatusCode < 201 {
		err = jsonUnmarshal(resp.Body, &out)
	}
	if resp.StatusCode >= 400 {
		errResp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		return nil, errors.New("Request to get collection " + id + " failed, json returned was: " + string(errResp))
	}
	out.Key = d.Key
	out.URI = d.URI
	out.Client = d.Client
	return &out, nil
}

func jsonUnmarshal(body io.Reader, v interface{}) error {
	data, _ := ioutil.ReadAll(body)
	return json.Unmarshal(data, v)
}
