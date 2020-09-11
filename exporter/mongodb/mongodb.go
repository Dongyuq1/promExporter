package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CmdResult struct {
}

type MtrData struct {
	Status         string      `json:"status"`
	Latency        float64     `json:"latency"`
	Url            string      `json:"url"`
	ServerLocation string      `json:"serverlocation"`
	Start          float64     `json:"start"`
	Params         string      `json:"params"`
	Result         []CmdResult `json:"result"`
	Type           string      `json:"type"`
}

type AmazonData struct {
	Amazon MtrData `json:"amazon"`
}

type MtrSensor struct {
	ISP         string `json:"isp"`
	Clientutc   int64  `json:"clientutc"`
	MtrDataByGW []MtrData
}

type MtrAllData struct {
	MtrDataByISP []MtrSensor
}

func queryMtrSensor(collection *mongo.Collection, timestamp int64, isp string) MtrSensor {
	start := timestamp - 180
	end := timestamp

	filter := bson.D{
		{"isp", isp},
		{"$and", bson.A{
			bson.D{{"clientutc", bson.M{"$gte": start}}},
			bson.D{{"clientutc", bson.M{"$lte": end}}},
		}},
	}
	original := make(map[string]interface{})
	var sensorData MtrSensor

	err := collection.FindOne(context.TODO(), filter).Decode(&original)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return sensorData
	} else {
		vstr, okstr := original["isp"].(string)
		if okstr {
			sensorData.ISP = vstr
		}

		vf64, okf64 := original["clientutc"].(float64)
		if okf64 {
			sensorData.Clientutc = int64(vf64)
		}

		a := original["sensorData"]
		jsondt, err := json.Marshal(a)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			return sensorData
		}

		var dat1 = AmazonData{}
		json.Unmarshal(jsondt, &dat1)
		dat1.Amazon.ServerLocation = "Amazon"
		dat1.Amazon.Url = "useast1-public-ubiservices.ubi.com"
		sensorData.MtrDataByGW = append(sensorData.MtrDataByGW, dat1.Amazon)

	}
	return sensorData
}

func QueryMtrAllData(timeStamp int64) MtrAllData {
	var result MtrAllData
	param := fmt.Sprintf("mongodb://10.196.12.230:27017")
	clientOptions := options.Client().ApplyURI(param)

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
		fmt.Println(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
		fmt.Println(err)
	}

	fmt.Println("Connected to MongoDB!")
	collection := client.Database("ccmsensor").Collection("mtr")

	result.MtrDataByISP = append(result.MtrDataByISP, queryMtrSensor(collection, timeStamp, "Hong Kong UBISOFT"))

	err = client.Disconnect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connection to MongoDB closed.")

	return result
}
