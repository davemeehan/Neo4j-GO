/** 
Neo4j REST client library written in GO
I took two methods from the Golang HTML package: EscapeString & escape. I appreciate it! Both of these are Copyright 2010 The Go Authors. All rights reserved.

Copyright (c) 2011, dave meehan
All rights reserved.
Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met: 
Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" 
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, 
THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR 
PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR 
CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, 
EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, 
PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; 
OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, 
WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR 
OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED 
OF THE POSSIBILITY OF SUCH DAMAGE.
**/

package neo4j

import (
	"net/http"
	"log"
	"errors"
	"encoding/json"
	"strings"
	"bytes"
	"strconv"
)

// general neo4j config
type Neo4j struct {
	Method     string // which http method
	StatusCode int    // last http status code received
	URL        string
}
type Error struct {
	List map[int]error
	Code int
}
// used when storing data returned from neo4j
type NeoTemplate struct {
	ID                  uint
	Relationships       string
	RelationshipsOut    string
	RelationshipsIn     string
	RelationshipsAll    string
	RelationshipsCreate string
	Data                map[string]interface{}
	Traverse            string
	Property            string
	Properties          string
	Self                string
	Extensions          map[string]interface{}
	Start               string        // relationships & traverse // returns both obj & string
	End                 string        // relationships & traverse // returns both obj & string
	Type                string        // relationships & traverse
	Indexed             string        // index related
	Length              string        // traverse framework
	Nodes               []interface{} // traverse framework
	TRelationships      []interface{} // traverse framework
}
// what chars to escape of course
const escapedChars = `&'<>"*[]:% `

func NewNeo4j(u string) (*Neo4j, error) {
	n := new(Neo4j)
	if len(u) < 1 {
		u = "http://127.0.0.1:7474/db/data"
	}
	n.URL = u
	_, err := n.send(u, "") // just a test to see if the connection is valid
	return n, err
}
/*
GetProperty(node id uint, name string) returns string of property value and any error raised as error
*/
func (this *Neo4j) GetProperty(id uint, name string) (string, error) {
	if len(name) < 1 {
		return "", errors.New("Property name must be at least 1 character.")
	}
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return "", err
	}
	this.Method = "get"
	body, err := this.send(node.Properties+"/"+name, "")
	if err != nil {
		return "", err
	}
	errorList := map[int]error{
		404: errors.New("Node or Property not found."),
		204: errors.New("No properties found."),
	}
	return body, this.NewError(errorList)
}
/*
GetProperties(node id uint)  returns a NeoTemplate struct and any errors raised as error
*/
func (this *Neo4j) GetProperties(id uint) (tmp *NeoTemplate, err error) {
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return tmp, err
	}
	this.Method = "get"
	body, err := this.send(node.Properties, "")
	if err != nil {
		return tmp, err
	}
	// pack json string into variable "data" so the json unmarshaler knows where to put it on struct type NeoTemplate
	jsonData, err := this.pack("data", body)
	if err != nil {
		return tmp, err
	}
	//convert json -> string and unmarshal -> NeoTemplate
	template, err := this.unmarshal(string(jsonData))
	if err != nil {
		return tmp, err
	}
	errorList := map[int]error{
		404: errors.New("Node or Property not found."),
		204: errors.New("No properties found."),
	}
	return template[0], this.NewError(errorList)
}
/*
SetProperty(node id uint, data map[string]string, replace bool) returns any error raised as error
typically replace should be false unless you wish to drop any other properties *not* specified in the data you sent to SetProperty
*/
func (this *Neo4j) SetProperty(id uint, data map[string]string, replace bool) error {
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return err
	}
	this.Method = "put"
	s, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if replace { // drop all properties on the node if they aren't specified in "data" ?
		_, err := this.send(node.Properties, string(s))
		if err != nil {
			return err
		}
	} else {
		for k, v := range data {
			k = strings.TrimSpace(k)                                     // strip leading & trailing whitespace from key
			_, err := this.send(node.Properties+"/"+k, strconv.Quote(v)) // wrap value in double quotes as neo4j expects
			if err != nil {
				return err
			}
		}
	}
	if err != nil {
		return err
	}
	errorList := map[int]error{
		404: errors.New("Node not found."),
		400: errors.New("Invalid data sent."),
	}
	return this.NewError(errorList)
}
/*
CreateProperty(node id uint, data map[string]string, replace bool) returns any errors raised as error
typically replace should be false unless you wish to drop any other properties *not* specified in the data you sent to CreateProperty
*/
func (this *Neo4j) CreateProperty(id uint, data map[string]string, replace bool) error {
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return err
	}
	s, err := json.Marshal(data)
	if err != nil {
		return err
	}
	this.Method = "put"
	if replace { // when replacing and dropping *ALL* values on node(not just new ones) we can simply pass in the entire json data set and neo4j will remove the old properties
		_, err := this.send(node.Properties, string(s))
		if err != nil {
			return err
		}
	} else { // if we are keeping the other properties on the node we must pass in new properties 1 at a time
		for k, v := range data {
			k = strings.TrimSpace(k)                                     // strip leading & trailing whitespace from key
			_, err := this.send(node.Properties+"/"+k, strconv.Quote(v)) // wrap value in double quotes as neo4j expects
			if err != nil {
				return err
			}
		}
	}
	errorList := map[int]error{
		404: errors.New("Node or Property not found."),
		400: errors.New("Invalid data sent."),
	}
	return this.NewError(errorList)
}
/*
DelProperty(node id uint, s string) returns any errors raised as error
pass in the id of the node and string as the the name/key of the property to delete
could be extended to also delete relationship properties as well
*/
func (this *Neo4j) DelProperty(id uint, s string) error {
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return err
	}
	this.Method = "delete"
	_, err = this.send(node.Properties+"/"+string(s), "")
	if err != nil {
		return err
	}
	errorList := map[int]error{
		404: errors.New("Node or Property not found."),
	}
	return this.NewError(errorList)
}
/*
DelNode(node id uint) returns any errors raised as error
*/
func (this *Neo4j) DelNode(id uint) error {
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return err
	}
	this.Method = "delete"
	_, err = this.send(node.Self, "")
	if err != nil {
		return err
	}
	errorList := map[int]error{
		404: errors.New("Node not found."),
		409: errors.New("Unable to delete node. May still have relationships."),
	}
	return this.NewError(errorList)
}
/*
CreateNode(data map[string]string) returns a NeoTemplate struct and any errors raised as error
*/
func (this *Neo4j) CreateNode(data map[string]string) (tmp *NeoTemplate, err error) {
	s, err := json.Marshal(data)
	if err != nil {
		return tmp, errors.New("Unable to Marshal Json data")
	}
	this.Method = "post"
	url := this.URL + "/node"
	body, err := this.send(url, string(s))
	if err != nil {
		return tmp, err
	}
	template, err := this.unmarshal(body) // json.Unmarshal wrapper with some type assertions etc
	if err != nil {
		return tmp, err
	}
	errorList := map[int]error{
		400: errors.New("Invalid data sent."),
	}
	return template[0], this.NewError(errorList)
}
/*
GetNode(id uint) returns a NeoTemplate struct and any errors raised as error
*/
func (this *Neo4j) GetNode(id uint) (tmp *NeoTemplate, err error) {
	if id < 1 {
		return tmp, errors.New("Invalid node id specified.")
	}
	this.Method = "get"
	url := this.URL + "/node/"
	body, err := this.send(url+strconv.FormatUint(id, 10), "") // convert uint -> string and send http request
	if err != nil {
		return tmp, err
	}
	template, err := this.unmarshal(body) // json.Unmarshal wrapper with some type assertions etc
	if err != nil {
		return tmp, err
	}
	errorList := map[int]error{
		404: errors.New("Node not found."),
	}
	return template[0], this.NewError(errorList)
}
/*
GetRelationshipsOnNode(node id uint, name string, direction string) returns an array of NeoTemplate structs containing relationship data and any errors raised as error
*/
func (this *Neo4j) GetRelationshipsOnNode(id uint, name string, direction string) (map[int]*NeoTemplate, error) {
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return nil, err
	}
	this.Method = "get"
	direction = strings.ToLower(direction)
	url := ""
	switch direction {
	case "in":
		url = node.RelationshipsIn
	case "out":
		url = node.RelationshipsOut
	case "all":
		fallthrough
	default:
		url = node.RelationshipsAll
	}
	body, err := this.send(url+"/"+name, "")
	if err != nil {
		return nil, err
	}
	template, err := this.unmarshal(body)
	if err != nil {
		return nil, err
	}
	errorList := map[int]error{
		404: errors.New("Node not found."),
	}
	return template, this.NewError(errorList)
}
/*
SetRelationship(relationship id uint, data map[string]string) returns any errors raised as error
id is the relationship id
*/
func (this *Neo4j) SetRelationship(id uint, data map[string]string) error {
	this.Method = "put"
	url := this.URL + "/relationship/"
	s, err := json.Marshal(data)
	if err != nil {
		return errors.New("Unable to Marshal Json data")
	}
	_, err = this.send(url+strconv.FormatUint(id, 10)+"/properties", string(s))
	if err != nil {
		return err
	}
	errorList := map[int]error{
		404: errors.New("Relationship not found."),
		400: errors.New("Invalid data sent."),
	}
	return this.NewError(errorList)
}
/*
DelRelationship(relationship id uint) returns any errors raised as error
you can pass in more than 1 id
*/
func (this *Neo4j) DelRelationship(id ...uint) error {
	this.Method = "delete"
	url := this.URL + "/relationship/"
	for _, i := range id {
		// delete each relationship for every id passed in
		_, err := this.send(url+strconv.FormatUint(i, 10), "")
		if err != nil {
			return err
		}
	}
	errorList := map[int]error{
		404: errors.New("Relationship not found."),
	}
	return this.NewError(errorList)
}
/*
CreateRelationship(src node id uint, dst node id uint, data map[string]string, relationship type string) returns any errors raised as error
*/
func (this *Neo4j) CreateRelationship(src uint, dst uint, data map[string]string, rType string) error {
	dstNode, err := this.GetNode(dst) // find properties for destination node so we can tie it into the relationship
	if err != nil {
		return err
	}
	srcNode, err := this.GetNode(src) // find properties for src node..
	if err != nil {
		return err
	}
	j := map[string]interface{}{} // empty map: keys are always strings in json, values vary
	j["to"] = dstNode.Self
	j["type"] = rType               // type of relationship
	j["data"] = map[string]string{} // empty array
	j["data"] = data                // add data to relationship
	s, err := json.Marshal(j)
	if err != nil {
		return errors.New("Unable to Marshal Json data")
	}
	this.Method = "post"
	_, err = this.send(srcNode.RelationshipsCreate, string(s)) // srcNode.RelationshipsCreate actually contains the full URL
	if err != nil {
		return err
	}
	errorList := map[int]error{
		404: errors.New("Node or 'to' node not found."),
		400: errors.New("Invalid data sent."),
	}
	return this.NewError(errorList)
}
/* 
SearchIdx(key string, value string, query string, category string, index type string) returns array of NeoTemplate structs and any errors raised as error
Lucene query lang: http://lucene.apache.org/java/3_1_0/queryparsersyntax.html
example query: the_key:the_* AND the_other_key:[1 TO 100]
if you specifiy a query, it will not search by key/value and vice versa
*/
func (this *Neo4j) SearchIdx(key string, value string, query string, cat string, idxType string) (map[int]*NeoTemplate, error) {
	url := this.URL + "/index/"
	if strings.ToLower(idxType) == "relationship" {
		url += "relationship"
	} else {
		url += "node"
	}
	url += "/" + cat
	if len(query) > 0 { // query set, ignore key/value pair
		url += "?query=" + this.EscapeString(query)
	} else { // search key, val
		url += "/" + strings.TrimSpace(key) + "/" + this.EscapeString(value)
	}
	this.Method = "get"
	body, err := this.send(url, "")
	if err != nil {
		return nil, err
	}
	template, err := this.unmarshal(body)
	if err != nil {
		return nil, err
	}
	errorList := map[int]error{
		400: errors.New("Invalid data sent."),
	}
	return template, this.NewError(errorList)
}

/* 
CreateIdx(node id uint, key string, value string, category string, index type string) returns any errors raised as error
*/
func (this *Neo4j) CreateIdx(id uint, key string, value string, cat string, idxType string) error {
	template, err := this.GetNode(id)
	if err != nil {
		return err
	}
	if len(cat) < 1 {
		idxType = "idx_nodes" // default, generic, index type
	}
	self := template.Self
	url := this.URL + "/index/"
	if strings.ToLower(idxType) == "relationship" {
		url += "relationship"
	} else {
		url += "node"
	}
	url += "/" + cat + "/" + key + "/" + value + "/"
	this.Method = "post"
	_, err = this.send(url, strconv.Quote(self)) // add double quotes around the node url as neo4j expects
	errorList := map[int]error{
		400: errors.New("Invalid data sent."),
	}
	return this.NewError(errorList)
}
/*
Traverse(node id uint, return type string, order string, uniqueness string, relationships map[string]string, depth int, prune map[string]string, filter map[string]string) returns array of NeoTemplate structs and any errors raised as error
*/
func (this *Neo4j) Traverse(id uint, returnType string, order string, uniqueness string, relationships map[string]string, depth int, prune map[string]string, filter map[string]string) (map[int]*NeoTemplate, error) {
	node, err := this.GetNode(id) // find properties for destination node
	if err != nil {
		return nil, err
	}
	j := map[string]interface{}{} // empty map: keys are always strings in json, values vary
	j["order"] = order
	j["max depth"] = depth
	j["uniqueness"] = uniqueness
	if relationships != nil {
		j["relationships"] = map[string]string{} // empty array
		j["relationships"] = relationships       // like: { "type": "KNOWS", "direction": "all" }
	}
	if prune != nil {
		j["prune evaluator"] = map[string]string{} // empty array
		j["prune evaluator"] = prune               // like: {  "language": "javascript", "body": "position.endNode().getProperty('date')>1234567;" }
	}
	if filter != nil {
		j["return filter"] = map[string]string{} // empty array
		j["return filter"] = filter              // like: { "language": "builtin","name": "all" }
	}
	s, err := json.Marshal(j)
	if err != nil {
		return nil, errors.New("Unable to Marshal Json data")
	}
	this.Method = "post"
	returnType = strings.ToLower(returnType)
	switch returnType { // really just a list of allowed values and anything else is replaced with "node"
	case "relationship":
	case "path":
	case "fullpath":
	case "node":
	default:
		returnType = "node"
	}
	url := strings.Replace(node.Traverse, "{returnType}", returnType, 1) // neo4j returns the traverse URL with the literal "{returnType}" at the end
	body, err := this.send(url, string(s))
	if err != nil {
		return nil, err
	}
	template, err := this.unmarshal(body)
	if err != nil {
		return nil, err
	}
	errorList := map[int]error{
		404: errors.New("Node not found."),
	}
	return template, this.NewError(errorList)
}

/* 
TraversePath(src node id uint, dst node id uint, relationships map[string]string, depth uint, algorithm string, paths bool) returns array of NeoTemplate structs and any errors raised as error
*/
func (this *Neo4j) TraversePath(src uint, dst uint, relationships map[string]string, depth uint, algo string, paths bool) (map[int]*NeoTemplate, error) {
	dstNode, err := this.GetNode(dst) // find properties for destination node
	if err != nil {
		return nil, err
	}
	srcNode, err := this.GetNode(src) // find properties for src node..
	if err != nil {
		return nil, err
	}
	j := map[string]interface{}{} // empty map: keys are always strings in json, values vary
	j["to"] = dstNode.Self
	j["max depth"] = depth
	j["algorithm"] = algo
	j["relationships"] = map[string]string{} // empty array
	j["relationships"] = relationships       // specify relationships like type: "KNOWS" direction: "all"
	s, err := json.Marshal(j)
	if err != nil {
		return nil, errors.New("Unable to Marshal Json data")
	}
	this.Method = "post"
	url := srcNode.Self
	if paths {
		url += "/paths"
	} else {
		url += "/path"
	}
	body, err := this.send(url, string(s))
	if err != nil {
		return nil, err
	}
	template, err := this.unmarshal(body)
	if err != nil {
		return nil, err
	}
	errorList := map[int]error{
		404: errors.New("No path found using current algorithm and parameters"),
	}
	return template, this.NewError(errorList)
}
/* shamelessly taken from golang html pkg */
func (this *Neo4j) EscapeString(s string) string {
	if strings.IndexAny(s, escapedChars) == -1 {
		return s
	}
	buf := bytes.NewBuffer(nil)
	this.escape(buf, s)
	return buf.String()
}
/* shamelessly taken from golang html pkg with a few minor updates */
func (this *Neo4j) escape(buf *bytes.Buffer, s string) {
	i := strings.IndexAny(s, escapedChars)
	for i != -1 {
		buf.WriteString(s[0:i])
		var esc string
		switch s[i] {
		case '%':
			esc = "%25"
		case '&':
			esc = "&amp;"
		case '\'':
			esc = "&apos;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '"':
			esc = "&quot;"
		case ' ':
			esc = "%20"
		case '*':
			esc = "%2A"
		case ':':
			esc = "%3A"
		case '[':
			esc = "%5B"
		case ']':
			esc = "%5D"
		default:
			panic("unrecognized escape character")
		}
		s = s[i+1:]
		buf.WriteString(esc)
		i = strings.IndexAny(s, escapedChars)
	}
	buf.WriteString(s)
}
// packs string literal into json object structure around variable "varName"
// data string should already be in json format
func (this *Neo4j) pack(name string, data string) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := json.Compact(buf, []byte("{ \""+name+"\": "+data+" } ")) // pkg data into new json string then compact() it onto our empty buffer
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}
func (this *Neo4j) send(url string, data string) (string, error) {
	var (
		resp *http.Response // http response
		buf  bytes.Buffer   // contains http response body
		err  error
	)
	if len(url) < 1 {
		url = this.URL + "node" // default path
	}
	client := new(http.Client)
	switch strings.ToLower(this.Method) { // which http method
	case "delete":
		req, e := http.NewRequest("DELETE", url, nil)
		if e != nil {
			err = e
			break
		}
		resp, err = client.Do(req)
	case "post":
		body := strings.NewReader(data)
		resp, err = client.Post(url,
			"application/json",
			body,
		)
	case "put":
		body := strings.NewReader(data)
		req, e := http.NewRequest("PUT", url, body)
		if e != nil {
			err = e
			break
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
	case "get":
		fallthrough
	default:
		resp, err = client.Get(url)
	}
	if err != nil {
		return "", err
	}
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}
	this.StatusCode = resp.StatusCode // the calling method should do more inspection with chkStatusCode() method and determine if the operation was successful or not.
	return buf.String(), nil
}
// this function unmarshals the individual node of data(or relationship etc). 
// called internally to build the dataset of records returned from neo4j
func (this *Neo4j) unmarshalNode(template map[string]interface{}) (*NeoTemplate, error) {
	var (
		data   interface{} // stores data from type assertion
		assert bool        // did the type assertion raise an err?
	)
	node := new(NeoTemplate)
	for k, v := range template { // loop result data
		data, assert = v.(map[string]interface{}) // type assertion
		if assert {
			switch vv := data.(type) { // switch on variable type so data/extensions are extracted properly
			case map[string]interface{}:
				switch k {
				case "data":
					node.Data = vv
				case "extensions":
					node.Extensions = vv
				}
			default:
				log.Printf("*Notice: Unknown type in JSON stream: %T from key: %v\n", vv, k)
			}
		} else { // to my knowledge neo4j is only going to pass strings and arrays so if map assertion failed above try an array instead
			data, assert = v.([]interface{}) // normal array?
			if assert {
				switch vv := data.(type) {
				case []interface{}:
					switch k {
					case "nodes":
						node.Nodes = vv
					case "relationships":
						node.TRelationships = vv
					}
				}
			} else { // if nothing else, it must be a string
				data, assert = v.(string)
				if assert {
					// copy string vars into node structure switch on key name
					switch k {
					case "self":
						node.Self, _ = data.(string) // cast it to a string with type assertion
						// "self" provides easy access to the ID property of the node(relationship, index,etc), we'll take advantage and axe it off right now
						selfSlice := strings.Split(string(node.Self), "/")         // slice string "Self" on each '/' char, -1 gets all instances
						id, atouiErr := strconv.ParseUint(selfSlice[len(selfSlice)-1], 10, 0) // and pull off the last part which is the ID then string -> uint
						if atouiErr != nil {
							return nil, atouiErr
						}
						node.ID = id
					case "traverse":
						node.Traverse, _ = data.(string)
					case "property":
						node.Property, _ = data.(string)
					case "properties":
						node.Properties, _ = data.(string)
					case "outgoing_relationships":
						node.RelationshipsOut, _ = data.(string)
					case "incoming_relationships":
						node.RelationshipsIn, _ = data.(string)
					case "all_relationships":
						node.RelationshipsAll, _ = data.(string)
					case "create_relationship":
						node.RelationshipsCreate, _ = data.(string)
					case "start": // relationships use this
						node.Start, _ = data.(string)
					case "end": // relationships use this
						node.End, _ = data.(string)
					case "type": // relationships use this
						node.Type, _ = data.(string)
					case "length":
						node.Length, _ = data.(string)
					case "indexed": // indices use this
						node.Indexed, _ = data.(string)
					}
				}
			}
		}
	}
	return node, nil
}
/*
json.Unmarshal wrapper
extracts json data into new interface and returns populated array of interfaces and any errors raised
*/
func (this *Neo4j) unmarshal(s string) (dataSet map[int]*NeoTemplate, err error) {
	var (
		templateNode map[string]interface{}   // blank interface for json.Unmarshal; used for node lvl data
		templateSet  []map[string]interface{} // array of blank interfaces for json.Unmarshal
	)
	dataSet = make(map[int]*NeoTemplate)           // make it ready for elements
	err = json.Unmarshal([]byte(s), &templateNode) // unmarshal json data into blank interface. the json pkg will populate with the proper data types
	if err != nil {                                // fails on multiple results
		err = json.Unmarshal([]byte(s), &templateSet) // if unable to unmarshal into single template, try an array of templates instead. If that fails, raise an error
		if err != nil {
			return nil, err
		}
		for _, v := range templateSet {
			data, err := this.unmarshalNode(v) // append NeoTemplate into the data set                             
			if err != nil {
				return nil, err
			}
			dataSet[len(dataSet)] = data // new array element containing data
		}
	} else {
		template, err := this.unmarshalNode(templateNode)
		if err != nil {
			return nil, err
		}
		dataSet[0] = template // just a single result
	}
	return
}
func (this *Neo4j) NewError(errorList map[int]error) error {
	if errorList != nil {
		errorList[500] = errors.New("Fatal Error 500.") // everything can return a 500 error
	}
	err := &Error{errorList, this.StatusCode}
	return err.check()
}
// checks the status code of the http response and returns an appropriate error(or not). 
func (this *Error) check() error {
	if this.List != nil {
		if this.List[this.Code] != nil {
			return this.List[this.Code]
		}
	}
	return nil // if error exists it was not defined in Error.List
}
