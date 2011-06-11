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
	"http"
	"log"
	"os"
	"json"
	"strings"
	"bytes"
	"strconv"
)

// general neo4j config
type Neo4j struct {
	Method         string // which http method
	BaseURL        string // probably a combination of the following vars
	ServerAddr     string
	ServerPort     string
	ServerBasePath string
	StatusCode     int                 // last http status code received
	Errors         map[string]os.Error // holds neo4j error strings
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
	Start               string        // relationships & traverse // might have to break into two vars because sometimes neo4j uses "start" to store a string, sometimes it is an object
	End                 string        // relationships & traverse // might have to break into two vars because sometimes neo4j uses "end" to store a string, sometimes it is an object
	Type                string        // relationships & traverse
	Indexed             string        // index related
	Length              string        // traverse framework
	Nodes               []interface{} // traverse framework
	TRelationships      []interface{} // traverse framework
}
// what chars to escape of course
const escapedChars = `&'<>"*[]: `
func New() (*Neo4j) {
	n := new(Neo4j)
	// just some defaults
	n.ServerAddr = "127.0.0.1"
	n.ServerPort = "7474"
	n.ServerBasePath = "/db/data"
	n.BaseURL = "http://" + n.ServerAddr + ":" + n.ServerPort + n.ServerBasePath

	n.Errors = make(map[string]os.Error, 21)
	n.Errors["UnknownStatus"] = os.NewError("Unknown Status Code returned.")
	n.Errors["500"] = os.NewError("Fatal Error 500.")
	n.Errors["404"] = os.NewError("Node, Property, Relationship or Index not found")

	// traverse
	n.Errors["TR404"] = os.NewError("Node or path not found.")
	n.Errors["TR204"] = os.NewError("No suitable path found.")

	// get property errors
	n.Errors["GP404"] = os.NewError("Node or Property not found.")
	n.Errors["GP204"] = os.NewError("No properties found.")

	// set property errors
	n.Errors["SP404"] = os.NewError("Node not found.")
	n.Errors["SP400"] = os.NewError("Invalid data sent.")

	// delete property errors
	n.Errors["DP404"] = os.NewError("Node or Property not found.")

	// set property errors
	n.Errors["SP400"] = os.NewError("Invalid data sent.")

	// create property errors
	n.Errors["CP404"] = os.NewError("Node or Property not found.")
	n.Errors["CP400"] = os.NewError("Invalid data sent.")

	// delete node errors
	n.Errors["DN404"] = os.NewError("Node not found.")
	n.Errors["DN409"] = os.NewError("Unable to delete node. May still have relationships.")

	// create relationship errors
	n.Errors["CR404"] = os.NewError("Node or 'to' node not found.")
	n.Errors["CR400"] = os.NewError("Invalid data sent.")

	// delete relationship errors
	n.Errors["DR404"] = os.NewError("Relationship not found.")

	// set relationship errors
	n.Errors["SR404"] = os.NewError("Relationship not found.")
	n.Errors["SR400"] = os.NewError("Invalid data sent.")

	// get relationship errors
	n.Errors["GR404"] = os.NewError("Node not found.")
	return n
}
/*
GetProperty(node id uint, name string) returns string of property value and any error raised as os.Error
*/
func (this *Neo4j) GetProperty(id uint, name string) (string, os.Error) {
	if len(name) < 1 {
		return "", os.NewError("Property name must be at least 1 character.")
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
	return body, this.chkStatusCode("gp")
}
/*
GetProperties(node id uint)  returns a NeoTemplate struct and any errors raised as os.Error
*/
func (this *Neo4j) GetProperties(id uint) (tmp *NeoTemplate, err os.Error) {
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
	return template[0], this.chkStatusCode("gp")
}
/*
SetProperty(node id uint, data map[string]string, replace bool) returns any error raised as os.Error
typically replace should be false unless you wish to drop any other properties *not* specified in the data you sent to SetProperty
*/
func (this *Neo4j) SetProperty(id uint, data map[string]string, replace bool) os.Error {
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
			k = strings.TrimSpace(k)                                  // strip leading & trailing whitespace from key
			_, err := this.send(node.Properties+"/"+k, strconv.Quote(v)) // wrap value in double quotes as neo4j expects
			if err != nil {
				return err
			}
		}
	}
	if err != nil {
		return err
	}
	return this.chkStatusCode("sp")
}
/*
CreateProperty(node id uint, data map[string]string, replace bool) returns any errors raised as os.Error
typically replace should be false unless you wish to drop any other properties *not* specified in the data you sent to CreateProperty
*/
func (this *Neo4j) CreateProperty(id uint, data map[string]string, replace bool) os.Error {
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
			k = strings.TrimSpace(k)                                  // strip leading & trailing whitespace from key
			_, err := this.send(node.Properties+"/"+k, strconv.Quote(v)) // wrap value in double quotes as neo4j expects
			if err != nil {
				return err
			}
		}
	}
	return this.chkStatusCode("cp")
}
/*
DelProperty(node id uint, s string) returns any errors raised as os.Error
pass in the id of the node and string as the the name/key of the property to delete
could be extended to also delete relationship properties as well
*/
func (this *Neo4j) DelProperty(id uint, s string) os.Error {
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return err
	}
	this.Method = "delete"
	_, err = this.send(node.Properties+"/"+string(s), "")
	if err != nil {
		return err
	}
	return this.chkStatusCode("dp")
}
/*
DelNode(node id uint) returns any errors raised as os.Error
*/
func (this *Neo4j) DelNode(id uint) os.Error {
	node, err := this.GetNode(id) // find properties for node
	if err != nil {
		return err
	}
	this.Method = "delete"
	_, err = this.send(node.Self, "")
	if err != nil {
		return err
	}
	return this.chkStatusCode("dn")
}
/*
CreateNode(data map[string]string) returns a NeoTemplate struct and any errors raised as os.Error
*/
func (this *Neo4j) CreateNode(data map[string]string) (tmp *NeoTemplate, err os.Error) {
	s, err := json.Marshal(data)
	if err != nil {
		return tmp, os.NewError("Unable to Marshal Json data")
	}
	this.Method = "post"
	url := this.BaseURL + "/node"
	body, err := this.send(url, string(s))
	if err != nil {
		return tmp, err
	}
	template, err := this.unmarshal(body) // json.Unmarshal wrapper with some type assertions etc
	if err != nil {
		return tmp, err
	}
	return template[0], this.chkStatusCode("cn") // creating a node returns a single result
}
/*
GetNode(id uint) returns a NeoTemplate struct and any errors raised as os.Error
*/
func (this *Neo4j) GetNode(id uint) (tmp *NeoTemplate, err os.Error) {
	if id < 1 {
		return tmp, os.NewError("Invalid node id specified.")
	}
	this.Method = "get"
	url := this.BaseURL + "/node/"
	body, err := this.send(url+strconv.Uitoa(id), "") // convert uint -> string and send http request
	if err != nil {
		return tmp, err
	}
	template, err := this.unmarshal(body) // json.Unmarshal wrapper with some type assertions etc
	if err != nil {
		return tmp, err
	}
	return template[0], this.chkStatusCode("gn")
}
/*
GetRelationshipsOnNode(node id uint, name string, direction string) returns an array of NeoTemplate structs containing relationship data and any errors raised as os.Error
*/
func (this *Neo4j) GetRelationshipsOnNode(id uint, name string, direction string) (map[int]*NeoTemplate, os.Error) {
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
	return template, this.chkStatusCode("gr")
}
/*
SetRelationship(relationship id uint, data map[string]string) returns any errors raised as os.Error
id is the relationship id
*/
func (this *Neo4j) SetRelationship(id uint, data map[string]string) os.Error {
	this.Method = "put"
	url := this.BaseURL + "/relationship/"
	s, err := json.Marshal(data)
	if err != nil {
		return os.NewError("Unable to Marshal Json data")
	}
	_, err = this.send(url+strconv.Uitoa(id)+"/properties", string(s))
	if err != nil {
		return err
	}
	return this.chkStatusCode("sr")
}
/*
DelRelationship(relationship id uint) returns any errors raised as os.Error
you can pass in more than 1 id
*/
func (this *Neo4j) DelRelationship(id ...uint) os.Error {
	this.Method = "delete"
	url := this.BaseURL + "/relationship/"
	for _, i := range id {
		// delete each relationship for every id passed in
		_, err := this.send(url+strconv.Uitoa(i), "")
		if err != nil {
			return err
		}
	}
	return this.chkStatusCode("dr")
}
/*
CreateRelationship(src node id uint, dst node id uint, data map[string]string, relationship type string) returns any errors raised as os.Error
*/
func (this *Neo4j) CreateRelationship(src uint, dst uint, data map[string]string, rType string) os.Error {
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
		return os.NewError("Unable to Marshal Json data")
	}
	this.Method = "post"
	_, err = this.send(srcNode.RelationshipsCreate, string(s)) // srcNode.RelationshipsCreate actually contains the full URL
	if err != nil {
		return err
	}
	return this.chkStatusCode("cr")
}
/* 
SearchIdx(key string, value string, query string, category string, index type string) returns array of NeoTemplate structs and any errors raised as os.Error
Lucene query lang: http://lucene.apache.org/java/3_1_0/queryparsersyntax.html
example query: the_key:the_* AND the_other_key:[1 TO 100]
if you specifiy a query, it will not search by key/value and vice versa
*/
func (this *Neo4j) SearchIdx(key string, value string, query string, cat string, idxType string) (map[int]*NeoTemplate, os.Error) {
	url := this.BaseURL + "/index/"
	if strings.ToLower(idxType) == "relationship" {
		url += "relationship"
	} else {
		url += "node"
	}
	url += "/" + cat
	if len(query) > 0 { // query set, ignore key/value pair
		url += "?query=" + this.EscapeString(query)
	} else { // default option, search key, val
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
	return template, this.chkStatusCode("si")
}

/* 
CreateIdx(node id uint, key string, value string, category string, index type string) returns any errors raised as os.Error
*/
func (this *Neo4j) CreateIdx(id uint, key string, value string, cat string, idxType string) os.Error {
	template, err := this.GetNode(id)
	if err != nil {
		return err
	}
	if len(cat) < 1 {
		idxType = "idx_nodes" // default, generic, index type
	}
	self := template.Self
	url := this.BaseURL + "/index/"
	if strings.ToLower(idxType) == "relationship" {
		url += "relationship"
	} else {
		url += "node"
	}
	url += "/" + cat + "/" + key + "/" + value + "/"
	this.Method = "post"
	_, err = this.send(url, strconv.Quote(self)) // add double quotes around the node url as neo4j expects
	return err
}

/*
Traverse(node id uint, return type string, order string, uniqueness string, relationships map[string]string, depth int, prune map[string]string, filter map[string]string) returns array of NeoTemplate structs and any errors raised as os.Error
*/
func (this *Neo4j) Traverse(id uint, returnType string, order string, uniqueness string, relationships map[string]string, depth int, prune map[string]string, filter map[string]string) (map[int]*NeoTemplate, os.Error) {
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
		return nil, os.NewError("Unable to Marshal Json data")
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
	return template, this.chkStatusCode("tr")
}

/* 
TraversePath(src node id uint, dst node id uint, relationships map[string]string, depth uint, algorithm string, paths bool) returns array of NeoTemplate structs and any errors raised as os.Error
*/
func (this *Neo4j) TraversePath(src uint, dst uint, relationships map[string]string, depth uint, algo string, paths bool) (map[int]*NeoTemplate, os.Error) {
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
		return nil, os.NewError("Unable to Marshal Json data")
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
	return template, this.chkStatusCode("tr")
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
// checks the status code of the http response and returns an appropriate error(or not). 
// We have to switch based on which method is calling this function because certain HTTP status codes have different meanings depending on the specific REST operation.
func (this *Neo4j) chkStatusCode(self string) os.Error {
	switch this.StatusCode {
	case 500: // fatal error for sure
		return this.Errors["500"]
	case 409: // inevitably failed
		switch strings.ToLower(self) {
		case "dn": // del node
			return this.Errors["DN409"]
		}
	case 404: // inevitably failed
		switch strings.ToLower(self) {
		case "cr": // create relationship
			return this.Errors["CR404"]
		case "dr": // del relationship
			return this.Errors["DR404"]
		case "sr": // set relationship
			return this.Errors["SR404"]
		case "gr": // get relationship
			return this.Errors["GR404"]
		case "cn": // create node
			return this.Errors["CN404"]
		case "dn": // delete node
			return this.Errors["DN404"]
		case "gn": // get node
			return this.Errors["GN404"]
		case "cp": // create property(ies)
			return this.Errors["CP404"]
		case "sp": // set property
			return this.Errors["SP404"]
		case "dp": // del property
			return this.Errors["DP404"]
		case "gp": // get property
			return this.Errors["GP404"]
		case "tr": // traverse
			return this.Errors["TR404"]
		default:
			return this.Errors["404"] // 404 is never good, return some sort of "not found" error
		}
	case 400:
		switch strings.ToLower(self) {
		case "cr": // create relationship
			return this.Errors["CR400"]
		case "dr": // del relationship
			return this.Errors["DR400"]
		case "sr": // set relationship
			return this.Errors["SR400"]
		case "cn": // create node
			return this.Errors["CN400"]
		case "cp": // create property(ies)
			return this.Errors["CP400"]
		case "sp": // set property
			return this.Errors["SP400"]
		case "gp": // get property
			return this.Errors["GP400"]
		}
	case 204:
		switch strings.ToLower(self) {
		case "gp": // get property
			return this.Errors["GP204"]
		case "tr": // traverse
			return this.Errors["TR204"]
		}
	case 201:
	case 200: // inevitably succeeded
	default:
		return this.Errors["UnknownStatus"]
	}
	return nil
}
// packs string literal into json object structure around variable "varName"
// data string should already be in json format
func (this *Neo4j) pack(name string, data string) ([]byte, os.Error) {
	buf := new(bytes.Buffer)
	err := json.Compact(buf, []byte("{ \""+name+"\": "+data+" } ")) // pkg data into new json string then compact() it onto our empty buffer
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}
func (this *Neo4j) send(url string, data string) (string, os.Error) {
	var (
		resp *http.Response // http response
		buf  bytes.Buffer   // contains http response body
		err  os.Error
	)
	if len(url) < 1 {
		url = this.BaseURL + "node" // default path
	}
	client := new(http.Client)
	switch strings.ToLower(this.Method) { // which http method
	case "delete":
		resp, err = client.Delete(url)
	case "post":
		body := strings.NewReader(data)
		resp, err = client.Post(url,
			"application/json",
			body,
		)
	case "put":
		body := strings.NewReader(data)
		if err != nil {
			return "", os.NewError("Unable to Marshal Json data")
		}
		resp, err = client.Put(url,
			"application/json",
			body,
		)
	case "get":
		fallthrough
	default:
		resp, _, err = client.Get(url)
	}
	if err != nil {
		return "", err
	}
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	this.StatusCode = resp.StatusCode // the calling method should do more inspection with chkStatusCode() method and determine if the operation was successful or not.
	return buf.String(), nil
}
// this function unmarshals the individual node of data(or relationship etc). 
// called internally to build the dataset of records returned from neo4j
func (this *Neo4j) unmarshalNode(template map[string]interface{}) (*NeoTemplate, os.Error) {
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
						selfSlice := strings.Split(string(node.Self), "/", -1)     // slice string "Self" on each '/' char, -1 gets all instances
						id, atouiErr := strconv.Atoui(selfSlice[len(selfSlice)-1]) // and pull off the last part which is the ID then string -> uint
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
func (this *Neo4j) unmarshal(s string) (dataSet map[int]*NeoTemplate, err os.Error) {
	var (
		templateNode map[string]interface{}   // blank interface for json.Unmarshal; used for node lvl data
		templateSet  []map[string]interface{} // array of blank interfaces for json.Unmarshal
	)
	dataSet = make(map[int]*NeoTemplate)         // make it ready for elements
	err = json.Unmarshal([]byte(s), &templateNode) // unmarshal json data into blank interface. the json pkg will populate with the proper data types
	if err != nil { // fails on multiple results
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
