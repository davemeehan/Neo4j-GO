package main

import (
	"github.com/masiulaniec/Neo4j-GO"
	"log"
)

func main() {
	neo, err := neo4j.NewNeo4j("http://localhost:7474/db/data")
	if err != nil {
		log.Printf("%v\n", err)
		return
	}
	node := map[string]string{
		"test1": "foo",
		"test2": "bar",
	}

	data, _ := neo.CreateNode(node)
	log.Printf("\nNode ID: %v\n", data.ID)
	self := data.ID

	data, _ = neo.GetNode(self)
	log.Printf("\nNode data: %v\n", data)

	err = neo.DelProperty((self + 5000), "test1") // will trigger an error unless you have over 5000 nodes in your db
	if err != nil {
		log.Printf("Del Property failed with: %v\n", err)
	} else {
		log.Printf("Deleted property from node\n")
	}

	node2 := map[string]string{
		"test3": "foo",
		"test4": "bar",
	}
	/* id(uint), data(map), remove bool (properties not specified?) */
	err = neo.CreateProperty(self, node2, false)
	if err != nil {
		log.Printf("Create Property failed with error: %v\n", err)
	} else {
		log.Printf("Properties updated for node: %v\n", self)
	}

	data, err = neo.GetProperties(self) // id(uint)
	if err != nil {
		log.Printf("Get Properties failed with error: %v\n", err)
	} else {
		log.Printf("Properties on node: %v\n", data.Data)
	}

	propVal, err := neo.GetProperty(self, "test3") // id(uint), name
	if err != nil {
		log.Printf("Get Property failed with error: %v\n", err)
	} else {
		log.Printf("Property on node: %v\n", propVal)
	}

	pdata := map[string]string{
		"test3": "foobar",
	}
	/* id(uint), data map[string]string, replace bool (remove properties not specified?) */
	err = neo.SetProperty(1, pdata, false)
	if err != nil {
		log.Printf("Set Prop failed with: %v\n", err)
	} else {
		log.Printf("Property updated.\n")
	}

	propVal, err = neo.GetProperty(self, "test4") // id(uint), name(string)
	if err != nil {
		log.Printf("Get Property failed with error: %v\n", err)
	} else {
		log.Printf("Property on node: %v\n", propVal)
	}

	ndata := map[string]string{
		"date": "May 26th 2011",
		"test": "true",
	}
	/* node id(uint), to node id(uint), data(map[string]string), type */
	err = neo.CreateRelationship(self, (self - 1), ndata, "KNOWS")
	if err != nil {
		log.Printf("Create Relationship failed with error: %v\n", err)
	} else {
		log.Printf("Relationship created for node: %v\n", self)
	}

	/* node id(uint), to node id(uint), data(map[string]string), type */
	err = neo.CreateRelationship(self, (self - 2), ndata, "KNOWS")
	if err != nil {
		log.Printf("Create Relationship failed with error: %v\n", err)
	} else {
		log.Printf("Relationship created for node: %v\n", self)
	}

	rdata := map[string]string{
		"date": "May 27th 2011",
		"test": "false",
	}
	/* idx key(string), idx value(string), idx category(string), idx type[node|relationship](string) */
	err = neo.CreateIdx((self - 1), "a_test", "testing1", "idx_type", "node")
	if err != nil {
		log.Printf("CreateIdx failed with error: %v\n", err)
	} else {
		log.Printf("Idx created.\n")
	}

	/* idx key(string), idx value(string), idx category(string), idx type[node|relationship](string) */
	err = neo.CreateIdx((self - 2), "a_test", "testing2", "idx_type", "node")
	if err != nil {
		log.Printf("CreateIdx failed with error: %v\n", err)
	} else {
		log.Printf("Idx created.\n")
	}

	/* idx key(string), idx value(string), lucene query(string), idx category(string), idx type[node|relationship](string) */
	dataSet, err := neo.SearchIdx("a_test", "testing1", "", "idx_type", "node")
	if err != nil {
		log.Printf("Search failed with error: %v\n", err)
	} else {
		for k, v := range dataSet { // loop the dataSet returned and print array
			log.Printf("searchidx %v is: %v\n", k, v.ID)
		}
	}

	/*
		idx key(string), idx value(string), lucene query(string), idx category(string), idx type[node|relationship](string)
		same query as above but using the lucene query language 
	*/
	dataSet, err = neo.SearchIdx("", "", "a_test:testing2", "idx_type", "node")
	if err != nil {
		log.Printf("Search failed with error: %v\n", err)
	} else {
		for k, v := range dataSet { // loop the dataSet returned and print array
			log.Printf("searchidx %v is: %v\n", k, v.ID)
		}
	}

	tdata := map[string]string{
		"type":      "KNOWS",
		"direction": "all",
	}
	/* src id(uint), dst id(uint), relationships(map[string]string), depth(uint), algorithm(string), paths(bool), find paths between? */
	dataSet, err = neo.TraversePath(self, (self - 2), tdata, 10, "shortestPath", true)
	if err != nil {
		log.Printf("TraversePath failed with error: %v\n", err)
	} else {
		for k, v := range dataSet { // loop the dataSet returned and print array key(int) and relationship ID
			log.Printf("TraversePath %v is: %v\n", k, v.TRelationships)
		}
	}

	/*
		src id(uint), dst id(uint), relationships(map[string]string), depth(uint), algorithm(string), paths(bool), find paths between? 
	*/
	dataSet, err = neo.TraversePath(self, (self - 2), tdata, 2, "shortestPath", true)
	if err != nil {
		log.Printf("Traverse failed with error: %v\n", err)
	} else {
		/* loop the dataSet returned and print array key(int) and relationship ID */
		for k, v := range dataSet {
			log.Printf("TraversePath %v is: %v\n", k, v.Nodes)
		}
	}
	filter := map[string]string{
		"language": "builtin",
		"name":     "all but start node",
	}
	/*
		Traverse(id uint, returnType string, order string, uniqueness string, relationships map[string]string, depth int, prune map[string]string, filter map[string]string)
		some possible values: uniqueness:[node|node path],  filter names:[all|all but start node] 
	*/
	dataSet, err = neo.Traverse(self, "node", "depth first", "node", nil, 2, nil, filter) //
	if err != nil {
		log.Printf("Traverse failed with error: %v\n", err)
	} else {
		for k, v := range dataSet { // loop the dataSet returned and print array key(int) and relationship ID
			log.Printf("Traverse %v is: %v\n", k, v.Self)
		}
	}

	// set & delete relationships on node
	dataSet, err = neo.GetRelationshipsOnNode(self, "KNOWS", "all") // id(uint), type string, direction string
	if err != nil {
		log.Printf("GetRelationshipsOnNode error: %v\n", err)
	} else {
		for _, v := range dataSet { // loop the dataSet returned and print array key(int) and relationship ID
			err = neo.SetRelationship(v.ID, rdata) // relationship id uint, map[string]string 
			if err != nil {
				log.Printf("Set relationship failed with error: %v\n", err)
			} else {
				log.Printf("Relationship properties updated.\n")
			}
			err = neo.DelRelationship(v.ID) // id ...uint  --relationship ids
			if err != nil {
				log.Printf("Del Relationship failed with error: %v\n", err)
			} else {
				log.Printf("Relationship deleted: %v\n", v.ID)
			}
		}
	}
	/*
		err = neo.DelNode(self)
		if err != nil {
			log.Printf("Del Node failed with: %v\n",err)
		} else {
			log.Printf("Deleted node\n")
		}
	*/
}
