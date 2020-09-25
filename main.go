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
	"strings"
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
	Likes int `json:"likes" bson:"likes" faker:"-"`
	Dislikes int `json:"dislikes" bson:"dislikes" faker:"-"`
}

type PageLink struct{
	Index int
	Link template.URL
}

type DataOnPage struct{
	Quantity int64
	Activities []Activity
	NowPage int
	Pages [5] PageLink
	RequestedActivity string
	CurrentFilter string
	FiltersPageLink template.URL
	MostLikedActivities[] Activity
	Filters [3] template.URL
	SearchName string
	SearchValue string
}

type SearchContainer struct{
	Search string
	Filters [3]string
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

func getNowTime() string{
	return time.Now().Format("2006.01.02 15:04:05")
}

func checkDirOfFilter(dir string) int{
	if dir == "desc"{
		return -1
	}
	return 1
}

func generateMD5(param string) string{
	return param + strconv.Itoa(time.Now().Second()) + strconv.Itoa(rand.Intn(128))
}

func getSearchContainerFromRequest(r *http.Request) SearchContainer{
	var filters [3]string
	for i:=1; i<=3; i++{
		filters[i-1] = r.URL.Query().Get("filter" + strconv.Itoa(i))
	}
	container := SearchContainer{
		Search: r.URL.Query().Get("search"),
		Filters: filters,
	}
	return container
}

func searchContainerToURLQuery(container *SearchContainer) string{
	url := make([]string, 0)
	if container.Search != ""{
		url = append(url, "search=" + container.Search)
	}
	cnt := 1
	for i:=0; i<len(container.Filters); i++{
		if container.Filters[i] != ""{
			url = append(url, "filter" + strconv.Itoa(cnt) + "=" + container.Filters[i])
			cnt++
		}
	}
	return strings.Join(url, "&")
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

func addActivity(param string) []mongo.InsertOneResult{
	activities := make([]Activity, 1)
	activities[0].Name = param
	timeNow := getNowTime()
	activities[0].DateOfCreation = timeNow
	activities[0].DateOfUpdate = timeNow
	activities[0].Flag = false
	activities[0].Likes = 0
	activities[0].Dislikes = 0
	return uploadActivitiesToDataBase(activities)
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

func editActivity(w http.ResponseWriter, r *http.Request){
	start := time.Now()
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
	fmt.Println("Time for edit activity = ", time.Since(start))
}

func createNewActivityEndpoint(w http.ResponseWriter, r *http.Request){
	start := time.Now()
	if r.Method == "POST"{
		name := r.FormValue("activity")
		file, header, err := r.FormFile("activity_image")
		if err != nil{
			fmt.Println(err.Error())
		}
		defer file.Close()
		idsOfInsertedActivities := addActivity(name)
		addImageOfActivity(idsOfInsertedActivities, file, header)
		fmt.Println(header.Filename)
		http.Redirect(w, r, "/TODO/page/1", 301)
	} else{
		temp, _ := template.ParseFiles("assets/createNewActivity.html")
		temp.Execute(w, "")
	}
	fmt.Println("Time for create new activity = ", time.Since(start))
}

func addImageOfActivity(idsOfInsertedActivities []mongo.InsertOneResult, file multipart.File, header *multipart.FileHeader){
	collection := client.Database("test").Collection("activities")
	for i:=0; i<len(idsOfInsertedActivities); i++{
		param := idsOfInsertedActivities[i].InsertedID.(primitive.ObjectID).Hex()
		imageSrc := generateMD5(param)
		if strings.Contains(header.Filename, ".png"){
			imageSrc = imageSrc + ".png"
		} else if strings.Contains(header.Filename, ".jpg"){
			imageSrc = imageSrc + ".jpg"
		} else if strings.Contains(header.Filename, ".jpeg"){
			imageSrc = imageSrc + ".jpeg"
		}
		collection.UpdateOne(
			context.TODO(),
			bson.D{{"_id", idsOfInsertedActivities[i].InsertedID}},
			bson.D{
				{"$set", bson.D{{"image_src", imageSrc}}},
			},
		)
		dir := ImageHostPath + imageSrc
		out, err := os.Create(dir)
		if err != nil {
			fmt.Println(err.Error())
		}
		_, err = io.Copy(out, file)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func calculateQuantityActivities(name string) int64{
	collection := client.Database("test").Collection("activities")
	quantity, _ := collection.EstimatedDocumentCount(context.TODO())
	if len(name) > 0{
		quantity, _ = collection.CountDocuments(context.TODO(), bson.D{{"name", name}})
	}
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

func renderDataOnPage(w http.ResponseWriter, data DataOnPage){
	temp, _ := template.ParseFiles("assets/index.html")
	temp.Execute(w, data)
}

func stableMainPage(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	filtersOfContent := make([][]string, 3)
	container := getSearchContainerFromRequest(r)
	currentFilter := searchContainerToURLQuery(&container)
	fmt.Println("search = ", container.Search)
	for i:=1; i<=3; i++{
		filtersOfContent[i-1] = strings.Split(r.URL.Query().Get("filter" + strconv.Itoa(i)), "-")
	}
	start := time.Now()
	page, err := strconv.Atoi(vars["id"])
	if err != nil{
		fmt.Println(err.Error())
	}
	maxPage := int(calculateQuantityActivities(container.Search) + 9) / 10
	if maxPage == 0{
		maxPage = 1
	}
	collection := client.Database("test").Collection("activities")
	skip := int64((page - 1) * 10)
	limit := int64(10)
	options := options.Find()
	options.SetSkip(skip)
	options.SetLimit(limit)
	options.SetSort(bson.D{{"flag", -1}})
	if len(filtersOfContent[0][0]) > 0{
		options.SetSort(bson.D{{"flag", -1},
			{filtersOfContent[0][0], checkDirOfFilter(filtersOfContent[0][1])},
			{filtersOfContent[1][0], checkDirOfFilter(filtersOfContent[1][1])},
			{filtersOfContent[2][0], checkDirOfFilter(filtersOfContent[2][1])}})
	}
	searchFilter := bson.D{{}}
	if len(container.Search) > 0{
		searchFilter = bson.D{{"name", container.Search}}
	}
	cursor, err := collection.Find(context.TODO(), searchFilter, options)
	if err != nil{
		fmt.Println(err.Error())
	}
	data := DataOnPage{
		Quantity:   calculateQuantityActivities(container.Search),
		Activities: readActivitiesFromCursor(cursor),
		NowPage: page,
		RequestedActivity: "Search",
		CurrentFilter: currentFilter,
	}
	if len(container.Search) > 0{
		data.SearchName = "search"
		data.SearchValue = container.Search
	}
	fmt.Println(container.Search)
	options.SetSkip(0)
	options.SetLimit(10)
	cursor, err = collection.Find(context.TODO(), bson.D{{}}, options)
	data.MostLikedActivities = readActivitiesFromCursor(cursor)
	for i:=-2; i<=2; i++{
		if page + i > 0 && maxPage >= page + i{
			data.Pages[2 + i].Index = page + i
			data.Pages[2 + i].Link = template.URL(strconv.Itoa(page + i)) + template.URL("?" + currentFilter)
		}
	}
	renderDataOnPage(w, data)
	duration := time.Since(start)
	fmt.Println("time for filters = ", duration)
	fmt.Println()
}

func uploadImage(param string, file multipart.File, header *multipart.FileHeader){
	collection := client.Database("test").Collection("activities")
	id, err := primitive.ObjectIDFromHex(param)
	if err != nil {
		fmt.Println(err.Error())
	}
	imageSrc := generateMD5(param)
	if strings.Contains(header.Filename, ".png"){
		imageSrc = imageSrc + ".png"
	} else if strings.Contains(header.Filename, ".jpg"){
		imageSrc = imageSrc + ".jpg"
	} else if strings.Contains(header.Filename, ".jpeg"){
		imageSrc = imageSrc + ".jpeg"
	}
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

func likeActivity(param string){
	id, err := primitive.ObjectIDFromHex(param)
	if err != nil{
		fmt.Println(err.Error())
	}
	collection := client.Database("test").Collection("activities")
	collection.UpdateOne(context.TODO(),
		bson.D{{"_id", id}},
		bson.D{{"$inc", bson.D{{"likes", 1}}}})
}

func dislikeActivity(param string){
	id, err := primitive.ObjectIDFromHex(param)
	if err != nil{
		fmt.Println(err.Error())
	}
	collection := client.Database("test").Collection("activities")
	collection.UpdateOne(context.TODO(),
		bson.D{{"_id", id}},
		bson.D{{"$inc", bson.D{{"dislikes", 1}}}})
}

func drawMainPage(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST"{
		action := r.FormValue("action")
		param := r.FormValue("activity_param")
		redirectPage := r.URL.String()
		if action == "add"{
			addActivity(param)
			http.Redirect(w, r, redirectPage,301)
		} else if action == "delete"{
			deleteActivity(param)
			http.Redirect(w, r, redirectPage,301)
		} else if action == "move_activity"{
			changePriorityOfActivity(w, param)
			http.Redirect(w, r, redirectPage,301)
		} else if action == "add_image" {
			file, handler, err := r.FormFile("activity_file")
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			defer file.Close()
			uploadImage(param, file, handler)
			http.Redirect(w, r, redirectPage, 301)
		} else if action == "like"{
			likeActivity(param)
			http.Redirect(w, r, redirectPage, 301)
		} else if action == "dislike"{
			dislikeActivity(param)
			http.Redirect(w, r, redirectPage, 301)
		}
	} else{
		stableMainPage(w, r)
	}
}

func uploadActivitiesToDataBase(activities []Activity) []mongo.InsertOneResult{
	collection := client.Database("test").Collection("activities")
	var idsOfInsertedActivities []mongo.InsertOneResult
	for i:=0; i<len(activities); i++{
		insertedActivity, err := collection.InsertOne(context.TODO(), activities[i])
		if err != nil{
			fmt.Println(err.Error())
		}
		idsOfInsertedActivities = append(idsOfInsertedActivities, *insertedActivity)
	}
	return idsOfInsertedActivities
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
	router := mux.NewRouter()
	router.HandleFunc("/createNewActivity", createNewActivityEndpoint)
	router.HandleFunc("/TODO/page/{id}", drawMainPage)
	router.HandleFunc("/editActivity", editActivity)
	router.HandleFunc("/TODO/createFakeData/{quantity}", createFakeData)
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets"))))
	http.Handle("/avatars/", http.StripPrefix("/avatars/", http.FileServer(http.Dir(ImageHostPath))))
	http.Handle("/", router)
	fmt.Println("Server is listening...")
	http.ListenAndServe(":9000", nil)
}