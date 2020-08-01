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
	"time"
)

type Activity struct{
	ID primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name string `json:"name" bson:"name"`
	Date string `json:"date" bson:"date"`
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

func addActivity(w http.ResponseWriter, r *http.Request){
	err := r.ParseMultipartForm(200000)
	if err != nil {
		log.Println(err)
	}
	result := r.FormValue("activity")
	action := r.FormValue("add")
	if len(action) > 0 {
		var activity Activity
		activity.Name = result
		activity.Date = time.Now().Format("2006.01.02 15:04:05")
		collection := client.Database("test").Collection("activities")
		_, err = collection.InsertOne(context.TODO(), activity)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	http.Redirect(w, r, "/TODO", 301)
}

func deleteActivity(w http.ResponseWriter, r *http.Request){
	err := r.ParseMultipartForm(200000)
	if err != nil {
		log.Println(err)
	}
	result := r.FormValue("activity")
	action := r.FormValue("delete")
	if len(action) > 0 {
		collection := client.Database("test").Collection("activities")
		_, err := collection.DeleteOne(context.TODO(), bson.M{"name": result})
		if err != nil{
			fmt.Println(err.Error())
		}
	}
	http.Redirect(w, r, "/TODO", 301)
}

func searchActivity(){

}

func stableMainPage(w http.ResponseWriter, r *http.Request){
	collection := client.Database("test").Collection("activities")
	cursor, err := collection.Find(context.TODO(), bson.M{})
	var data []Activity
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		var activity Activity
		if err = cursor.Decode(&activity); err != nil {
			log.Fatal(err)
		}
		data = append(data, activity)
	}
	temp, _ := template.ParseFiles("assets/index.html")
	temp.Execute(w, data)
}

func drawMainPage(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST"{
		addActivity(w, r)
		deleteActivity(w, r)
	} else{
		stableMainPage(w, r)
	}
}

func main(){
	fmt.Println(time.Now().Format("2006.01.02 15:04:05"))
	http.HandleFunc("/TODO", drawMainPage)
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets"))))
	fmt.Println("Server is listening...")
	http.ListenAndServe(":9000", nil)
}