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
	fmt.Println("Connection to MongoDB = true")
	return client, nil
}

func addActivity(param string){
	var activity Activity
	activity.Name = param
	activity.Date = time.Now().Format("2006.01.02 15:04:05")
	collection := client.Database("test").Collection("activities")
	_, err := collection.InsertOne(context.TODO(), activity)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func deleteActivity(param string){
	collection := client.Database("test").Collection("activities")
	_, err := collection.DeleteOne(context.TODO(), bson.D{{"name", param}})
	if err != nil{
		fmt.Println(err.Error())
	}
}

func searchActivity(w http.ResponseWriter, param string){
	collection := client.Database("test").Collection("activities")
	answer, _ := collection.Find(context.TODO(), bson.D{{"name", param}})
	defer answer.Close(context.TODO())
	activities := readActivitiesFromCursor(answer)
	renderList(w, activities)
}

func editActivity(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST"{
		url := r.URL.String()
		activity :=r.FormValue("activity")
		objectId, err := primitive.ObjectIDFromHex(url[31:55])
		if err != nil{
			fmt.Println(err.Error())
		}
		collection := client.Database("test").Collection("activities")
		date := time.Now().Format("2006.01.02 15:04:05")
		_, err = collection.UpdateOne(
			context.TODO(),
			bson.D{{"_id", objectId}},
			bson.D{
				{"$set", bson.D{{"name", activity}}},
				{"$set", bson.D{{"date", date}}},
			},
		)
		if err != nil{
			fmt.Println(err.Error())
		}
		http.Redirect(w, r, "/TODO",301)
	} else{
		temp, _ := template.ParseFiles("assets/editActivity.html")
		temp.Execute(w, "")
	}
}

func readActivitiesFromCursor(cursor *mongo.Cursor) []Activity{
	var data []Activity
	for cursor.Next(context.TODO()) {
		var activity Activity
		if err := cursor.Decode(&activity); err != nil {
			log.Fatal(err)
		}
		data = append(data, activity)
	}
	return data
}

func renderList(w http.ResponseWriter, activities []Activity){
	temp, _ := template.ParseFiles("assets/index.html")
	temp.Execute(w, activities)
}

func stableMainPage(w http.ResponseWriter, r *http.Request){
	collection := client.Database("test").Collection("activities")
	cursor, err := collection.Find(context.TODO(), bson.M{})
	defer cursor.Close(context.TODO())
	if err != nil{
		fmt.Println(err.Error())
	}
	data := readActivitiesFromCursor(cursor)
	renderList(w, data)
}

func drawMainPage(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST"{
		action := r.FormValue("action")
		activity := r.FormValue("activity")
		if action == "search"{
			searchActivity(w, activity)
		} else if action == "add"{
			addActivity(activity)
			http.Redirect(w, r, "/TODO",301)
		} else if action == "delete"{
			deleteActivity(activity)
			http.Redirect(w, r, "/TODO",301)
		}
	} else{
		stableMainPage(w, r)
	}
}

func main(){
	fmt.Println(time.Now().Format("2006.01.02 15:04:05"))
	http.HandleFunc("/TODO", drawMainPage)
	http.HandleFunc("/editActivity", editActivity)
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets"))))
	fmt.Println("Server is listening...")
	http.ListenAndServe(":9000", nil)
}
