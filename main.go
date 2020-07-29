package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"log"
	"net/http"
)

type Activity struct{
	ID primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name string `json:"name" bson:"name"`
}

var(
	client, _ = initMongo()
)

func initMongo() (*mongo.Client, error){
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB = true")
	return client, nil
}

func drawMainPage(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST"{
		err := r.ParseMultipartForm(200000)
		if err != nil {
			log.Println(err)
		}
		result := r.FormValue("activity")
		var activity Activity
		activity.Name = result
		collection := client.Database("test").Collection("activities")
		_, err = collection.InsertOne(context.TODO(), activity)
		if err != nil{
			fmt.Println(err.Error())
		}
		http.Redirect(w, r, "/TODO", 301)
	} else{
		collection := client.Database("test").Collection("activities")
		cursor, err := collection.Find(context.TODO(), bson.M{})
		var activity Activity
		var activities []string
		defer cursor.Close(context.TODO())
		for cursor.Next(context.TODO()) {
			if err = cursor.Decode(&activity); err != nil {
				log.Fatal(err)
			}
			activities = append(activities, activity.Name)
		}
		temp, _ := template.ParseFiles("assets/index.html")
		temp.Execute(w, activities)
	}
}

func main(){
	http.HandleFunc("/TODO", drawMainPage)
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets"))))
	fmt.Println("Server is listening...")
	http.ListenAndServe(":9000", nil)
}
