package main

import (
	"context"
	"fmt"
	"log"
	"root/blog/blogpb"

	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Blog Client")

	//Establishing the connection with server
	//'cc' contains the connection with server
	cc, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error establishing connection with server: %v", err)
	}

	//Creating a new Blog Service Client, passing in the connection with server
	c := blogpb.NewBlogServiceClient(cc)

	/***************************************************************************************************/

	//Request to create a blog:
	req := &blogpb.CreateBlogRequest{
		Blog: &blogpb.Blog{
			AuthodId: "Apurv",
			Title:    "My First Blog",
			Content:  "My first blog contents",
		},
	}

	createBlogRes, createBlogErr := c.CreateBlog(context.Background(), req)

	if createBlogErr != nil {
		fmt.Printf("Error occurred while creating the blog")
	}

	fmt.Printf("Blog created successfully, response from server: %v", createBlogRes)

	/***************************************************************************************************/

	//Request to read the blog
	fmt.Println("Reading the blog..")

	//Making a ReadBlogRequest with wrong blog id to check the error handling of server
	_, readBlogError1 := c.ReadBlog(context.Background(), &blogpb.ReadBlogRequest{BlogId: "sdnflsdnfkl"})

	if readBlogError1 != nil {
		fmt.Printf("Error occurred while reading the blog: %v \n", readBlogError1)
	}

	//Making a ReadBlogRequest with correct blog id to actually read the blog
	//Extracting the BlogID from the blog we created in CreateBlogRequest:

	blogID := createBlogRes.GetBlog().GetId()

	readBlogRes, readBlogError2 := c.ReadBlog(context.Background(), &blogpb.ReadBlogRequest{BlogId: blogID})

	if readBlogError2 != nil {
		fmt.Printf("Some error occurred: %v \n", readBlogError2)
	}

	fmt.Printf("The read blog response is: \n %v \n", readBlogRes)

	/***************************************************************************************************/

	//Request to Update the Blog:
	fmt.Println("Update blog request")

	//Using the same blog id as the one in read blog request
	updateBlogReq := &blogpb.UpdateBlogRequest{
		Blog: &blogpb.Blog{
			Id:       blogID,
			AuthodId: "Apurv Sawant",
			Title:    "My Second Blog",
			Content:  "My second Blog content",
		},
	}

	//Making call to UpdateBlog
	updateBlogRes, updateBlogError := c.UpdateBlog(context.TODO(), updateBlogReq)
	if updateBlogError != nil {
		fmt.Printf("Some error occurred: %v", updateBlogError)
	}

	//Printing the response to console:
	fmt.Printf("Updated blog with blog id: %v successfully \n", blogID)
	fmt.Printf("The Update Blog response from server is: %v", updateBlogRes)

	/***************************************************************************************************/

	fmt.Println("Making a delete blog request")

	deleteBlogReq := &blogpb.DeleteBlogRequest{
		BlogId: blogID,
	}

	deleteBlogRes, deleteBlogErr := c.DeleteBlog(context.Background(), deleteBlogReq)
	if deleteBlogErr != nil {
		fmt.Printf("Could not delete the blog: %v", deleteBlogErr)
	}

	fmt.Printf("Deleted blog with blog id: %v successfully \n", blogID)
	fmt.Printf("Delete Blog Response from server: %v \n", deleteBlogRes)
}
