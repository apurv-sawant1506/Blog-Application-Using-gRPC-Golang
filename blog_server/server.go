package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"root/blog/blogpb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var collection *mongo.Collection

type server struct{}

//Creating a item which will map to a document in MongoDB
//BSON is a Binary JSON. It is a data storage format originated at MongoDB in 2009

// use type primitive.ObjectId for ID
type blogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id"`
	Content  string             `bson:"content"`
	Title    string             `bson:"title"`
}

func (*server) CreateBlog(ctx context.Context, req *blogpb.CreateBlogRequest) (*blogpb.CreateBlogResponse, error) {
	fmt.Println("Creating a blog..")

	//extracting Blog from request
	blog := req.GetBlog()

	//Creating struct blogItem variable to send to MongoDB
	//We are not giving an id to the document
	//If no id is present, an ID will be automatically given while marshalling
	data := blogItem{
		AuthorID: blog.GetAuthodId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}

	//InsertOne(..) method is used to insert a single document in mongodb
	res, err := collection.InsertOne(context.Background(), data)

	//If some error occurs, we return response as nil and a proper error
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal Error while inserting document in database: %v", err),
		)
	}

	//To retrieve the id given to the object
	oid, ok := res.InsertedID.(primitive.ObjectID)

	//If not ok then we deal with the error
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Some internal error occurred while getting ObjectID"),
		)
	}

	//Everything is fine if we reach this point, hence we will return proper response to the client
	return &blogpb.CreateBlogResponse{
		Blog: &blogpb.Blog{
			Id:       oid.Hex(),
			AuthodId: blog.GetAuthodId(),
			Title:    blog.GetTitle(),
			Content:  blog.GetContent(),
		},
	}, nil
}

func (*server) ReadBlog(ctx context.Context, req *blogpb.ReadBlogRequest) (*blogpb.ReadBlogResponse, error) {
	fmt.Println("Read Blog Request")

	blogID := req.GetBlogId()

	oid, err := primitive.ObjectIDFromHex(blogID)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	//create an empty struct
	data := &blogItem{}

	filter := bson.M{"_id": oid}

	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog with specified ID: %v", err),
		)
	}

	return &blogpb.ReadBlogResponse{
		Blog: &blogpb.Blog{
			Id:       data.ID.Hex(),
			AuthodId: data.AuthorID,
			Content:  data.Content,
			Title:    data.Title,
		},
	}, nil
}

//Implementing UpdateBlog function
func (*server) UpdateBlog(ctx context.Context, req *blogpb.UpdateBlogRequest) (*blogpb.UpdateBlogResponse, error) {
	fmt.Println("Update Blog Request")

	blog := req.GetBlog()

	oid, err := primitive.ObjectIDFromHex(blog.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	//Creating an empty data struct
	data := &blogItem{}

	//Creating the filter:
	filter := bson.M{"_id": oid}

	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog with specified ID: %v", err),
		)
	}

	//updating the data struct:
	data.AuthorID = blog.GetAuthodId()
	data.Content = blog.GetContent()
	data.Title = blog.GetTitle()

	//Now we update the blog in the database:

	_, UpdateErr := collection.ReplaceOne(context.Background(), filter, data)
	if UpdateErr != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Error occurred while updating the blog: %v", UpdateErr),
		)
	}

	//Now that everything is okay, we return the UpdateBlogResponse to the user:
	return &blogpb.UpdateBlogResponse{
		Blog: &blogpb.Blog{
			Id:       oid.Hex(),
			AuthodId: data.AuthorID,
			Title:    data.Title,
			Content:  data.Content,
		},
	}, nil
}

//Implementing DeleteBlog
func (*server) DeleteBlog(ctx context.Context, req *blogpb.DeleteBlogRequest) (*blogpb.DeleteBlogResponse, error) {
	fmt.Println("Delete Blog Request")

	blogID := req.GetBlogId()

	oid, err := primitive.ObjectIDFromHex(blogID)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Some error occurred: %v", err),
		)
	}

	filter := bson.M{"_id": oid}

	res, deleteBlogErr := collection.DeleteOne(context.Background(), filter)

	if deleteBlogErr != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Something went wrong: %v", deleteBlogErr),
		)
	}

	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find document with specified ID: %v", deleteBlogErr),
		)
	}

	fmt.Println("Deleted blog successfully")

	//Now that everything is okay, we sent the appropriate response to the client:
	return &blogpb.DeleteBlogResponse{
		BlogId: oid.Hex(),
	}, nil
}

func main() {

	//If we crash the go code, we get the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("Starting blog service")

	//Connecting to MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())

	//Creating a database and a collection in that database:
	collection = client.Database("mydb").Collection("blog")

	//Establishing a connection, using specified protocols
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Error occured while starting the server: %v", err)
	}

	//Creating a new server
	s := grpc.NewServer()

	//Registering the server to Blog Service
	blogpb.RegisterBlogServiceServer(s, &server{})

	//Starting the server
	error := s.Serve(lis)
	if error != nil {
		log.Fatalf("Failed to serve: %v", error)
	}

	//Closing the server properly, using 'Ctrl+C' keyboard interrupt

	// Wait for Control C to exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// Block until a signal is received
	<-ch
	// First we close the connection with MongoDB:
	fmt.Println("Closing MongoDB Connection")
	// client.Disconnect(context.TODO())
	if err := client.Disconnect(context.TODO()); err != nil {
		log.Fatalf("Error on disconnection with MongoDB : %v", err)
	}
	// Second step : closing the listener
	fmt.Println("Closing the listener")
	if err := lis.Close(); err != nil {
		log.Fatalf("Error on closing the listener : %v", err)
	}
	// Finally, we stop the server
	fmt.Println("Stopping the server")
	s.Stop()
	fmt.Println("End of Program")
}
