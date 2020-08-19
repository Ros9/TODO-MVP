package main

import (
	"context"
	"fmt"
	"github.com/bxcodec/faker/v3"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Activity struct{
	ID primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty" faker:"-"`
	SimpleID string `json:"simple_id" faker:"-"`
	Name string `json:"name" bson:"name" faker:"len=10"`
	DateOfCreation string `json:"date_of_creation" bson:"date_of_creation" faker:"-"`
	DateOfUpdate string `json:"date_of_update" bson:"date_of_update" faker:"-"`
	Flag bool `json:"flag" bson:"flag" faker:"-"`
	ImageSrc string `json:"image_src" bson:"image_src" faker:"-"`
}

type DataOnPage struct{
	Quantity int64
	Activities []Activity
	NowPage int
	Pages [5]int
}

const ImageHostPath = "/home/abdrasul/Development/images/"

var(
	client, _ = initMongo()
)

func getMin(x, y int) int{
	if x > y{
		return y
	}
	return x
}

func getMax(x, y int) int{
	if x > y{
		return x
	}
	return y
}

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

func getNowTime() string{
	return time.Now().Format("2006.01.02 15:04:05")
}

func addActivity(param string){
	activities := make([]Activity, 1)
	activities[0].Name = param
	timeNow := getNowTime()
	activities[0].DateOfCreation = timeNow
	activities[0].DateOfUpdate = timeNow
	activities[0].Flag = false
	uploadActivitiesToDataBase(activities)
}

func deleteActivity(param string){
	id, err := primitive.ObjectIDFromHex(param)
	if err != nil{
		fmt.Println(err.Error())
	}
	collection := client.Database("test").Collection("activities")
	_, err = collection.DeleteOne(context.TODO(), bson.D{{"_id", id}})
	if err != nil{
		fmt.Println(err.Error())
	}
}

func searchActivity(w http.ResponseWriter, param string){
	collection := client.Database("test").Collection("activities")
	answer, _ := collection.Find(context.TODO(), bson.D{{"name", param}})
	defer answer.Close(context.TODO())
	data := DataOnPage{
		Quantity: calculateQuantityActivities(),
		Activities: readActivitiesFromCursor(answer),
	}
	renderData(w, data)
}

func editActivity(w http.ResponseWriter, r *http.Request){
	id, _ := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	collection := client.Database("test").Collection("activities")
	if r.Method == "POST"{
		text := r.FormValue("activity")
		date := getNowTime()
		_, err := collection.UpdateOne(
			context.TODO(),
			bson.D{{"_id", id}},
			bson.D{
				{"$set", bson.D{{"name", text}}},
				{"$set", bson.D{{"date_of_update", date}}},
			},
		)
		if err != nil{
			fmt.Println(err.Error())
		}
		http.Redirect(w, r, "/TODO/page/1",301)
	} else{
		var data Activity
		err := collection.FindOne(context.TODO(), bson.D{{"_id", id}}).Decode(&data)
		if err != nil{
			http.Redirect(w, r, "/TODO/page/1",301)
			return
		}
		temp, _ := template.ParseFiles("assets/editActivity.html")
		temp.Execute(w, data)
	}
}

func calculateQuantityActivities() int64{
	collection := client.Database("test").Collection("activities")
	quantity, _ := collection.EstimatedDocumentCount(context.TODO())
	return quantity
}

func changePriorityOfActivity(w http.ResponseWriter, param string){
	id, err := primitive.ObjectIDFromHex(param)
	if err != nil{
		fmt.Println(err.Error())
	}
	collection := client.Database("test").Collection("activities")
	cursor, err := collection.Find(context.TODO(), bson.D{{"_id", id}})
	if err != nil{
		fmt.Println(err.Error())
	}
	activities := make([]Activity, 1)
	activities = readActivitiesFromCursor(cursor)
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.D{{"_id", id}},
		bson.D{
			{"$set", bson.D{{"flag", !activities[0].Flag}}},
		},
	)
	if err != nil{
		fmt.Println(err.Error())
	}
}

func readActivitiesFromCursor(cursor *mongo.Cursor) []Activity{
	var data []Activity
	for cursor.Next(context.TODO()) {
		var activity Activity
		if err := cursor.Decode(&activity); err != nil {
			log.Fatal(err)
		}
		activity.SimpleID = activity.ID.Hex()
		data = append(data, activity)
	}
	return data
}

func renderData(w http.ResponseWriter, data DataOnPage){
	temp, _ := template.ParseFiles("assets/index.html")
	temp.Execute(w, data)
}

func renderDataOnPage(w http.ResponseWriter, data DataOnPage, param string){
	temp, _ := template.ParseFiles("assets/index.html")
	temp.Execute(w, data)
}

func stableMainPage(w http.ResponseWriter, r *http.Request, param string){
	start := time.Now()
	page, err := strconv.Atoi(param)
	if err != nil{
		fmt.Println(err.Error())
	}
	maxPage := int(calculateQuantityActivities() + 9) / 10
	if page > maxPage {
		redirectPage := "/TODO/page/" + strconv.Itoa(maxPage)
		http.Redirect(w, r, redirectPage, 301)
	}
	collection := client.Database("test").Collection("activities")
	skip := int64((page - 1) * 10)
	limit := int64(10)
	options := options.Find()
	options.SetSkip(skip)
	options.SetLimit(limit)
	cursor, err := collection.Find(context.TODO(), bson.M{}, options)
	defer cursor.Close(context.TODO())
	if err != nil{
		fmt.Println(err.Error())
	}
	data := DataOnPage{
		Quantity:   calculateQuantityActivities(),
		Activities: readActivitiesFromCursor(cursor),
		NowPage: page,
	}
	for i:=2; i>=0; i--{
		if page - i > 0{
			data.Pages[2-i] = page - i
		}
	}
	for i:=1; i<=2; i++{
		if maxPage >= page + i{
			data.Pages[2+i] = page + i
		}
	}
	renderDataOnPage(w, data, param)
	duration := time.Since(start)
	fmt.Println(duration)
}

func generateMD5(param string) string{
	return param + strconv.Itoa(time.Now().Second()) + strconv.Itoa(rand.Intn(128))
}

func uploadImage(param string, file multipart.File, handler *multipart.FileHeader){
	collection := client.Database("test").Collection("activities")
	id, err := primitive.ObjectIDFromHex(param)
	if err != nil {
		fmt.Println(err.Error())
	}
	imageSrc := generateMD5(param) + handler.Filename
	collection.UpdateOne(
		context.TODO(),
		bson.D{{"_id", id}},
		bson.D{
			{"$set", bson.D{{"image_src", imageSrc}}},
		},
	)
	dir := ImageHostPath + imageSrc
	out, err := os.Create(dir)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer out.Close()
}

func drawMainPage(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	if r.Method == "POST"{
		action := r.FormValue("action")
		param := r.FormValue("activity_param")
		redirectPage := "/TODO/page/" + vars["id"]
		if action == "search"{
			searchActivity(w, param)
		} else if action == "add"{
			addActivity(param)
			http.Redirect(w, r, redirectPage,301)
		} else if action == "delete"{
			deleteActivity(param)
			http.Redirect(w, r, redirectPage,301)
		} else if action == "move_activity"{
			changePriorityOfActivity(w, param)
			http.Redirect(w, r, redirectPage,301)
		} else if action == "add_image"{
			file, handler, err := r.FormFile("activity_file")
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			defer file.Close()
			uploadImage(param, file, handler)
			http.Redirect(w, r, redirectPage,301)
		}
	} else{
		stableMainPage(w, r, vars["id"])
	}
}

func uploadActivitiesToDataBase(activities []Activity){
	collection := client.Database("test").Collection("activities")
	for i:=0; i<len(activities); i++{
		_, err := collection.InsertOne(context.TODO(), activities[i])
		if err != nil{
			fmt.Println(err.Error())
			return
		}
	}
}

func createFakeData(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	quantity, err := strconv.Atoi(vars["quantity"])
	if err != nil{
		fmt.Println(err.Error())
		return
	}
	activities := make([]Activity, quantity)
	for i:=0; i<quantity; i++{
		var activity Activity
		err := faker.FakeData(&activity)
		if err != nil{
			fmt.Println(err.Error())
			return
		}
		activity.DateOfCreation = getNowTime()
		activity.DateOfUpdate = getNowTime()
		activities[i] = activity
	}
	uploadActivitiesToDataBase(activities)
	http.Redirect(w, r, "/TODO/page/1",301)
}

func main(){
	fmt.Println(calculateQuantityActivities())
	router := mux.NewRouter()
	router.HandleFunc("/TODO/page/{id}", drawMainPage)
	router.HandleFunc("/editActivity", editActivity)
	router.HandleFunc("/TODO/createFakeData/{quantity}", createFakeData)
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets"))))
	http.Handle("/avatars/", http.StripPrefix("/avatars/", http.FileServer(http.Dir(ImageHostPath))))
	http.Handle("/", router)
	fmt.Println("Server is listening...")
	http.ListenAndServe(":9000", nil)
}
