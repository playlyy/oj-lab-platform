package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	gormAgent "github.com/OJ-lab/oj-lab-services/src/core/agent/gorm"
	minioAgent "github.com/OJ-lab/oj-lab-services/src/core/agent/minio"
	"github.com/OJ-lab/oj-lab-services/src/service/mapper"
	"github.com/OJ-lab/oj-lab-services/src/service/model"
	yaml "gopkg.in/yaml.v2"

	"github.com/minio/minio-go/v7"
)

var ctx = context.Background()

func main() {
	db := gormAgent.GetDefaultDB()
	minioClient := minioAgent.GetMinioClient()

	log.Printf("%#v\n", minioClient) // minioClient is now set up
	bucketName := minioAgent.GetBucketName()

	err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	// Read package files
	// Search Problem under packagePath
	// 1. parse problem path as `slug`,
	//    parse problem.yaml's name as `title`,
	//    parse problem.md as description.
	// 2. insert object into minio storage.
	packagePath := "tests/data/packages/icpc"
	title := ""
	slug := ""
	filepath.Walk(packagePath, func(path string, info fs.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("file info is nil")
		}
		if info.IsDir() {
			return nil
		}
		relativePath := strings.Replace(path, packagePath, "", 1)
		println("relativePath: ", relativePath)
		if filepath.Base(relativePath) == "problem.yaml" {
			resultMap := make(map[string]interface{})
			yamlFile, err := os.ReadFile(path)
			if err != nil {
				log.Println(err)
			}
			err = yaml.Unmarshal(yamlFile, &resultMap)
			if err != nil {
				log.Printf("Unmarshal: %v\n", err)
			}
			title = resultMap["name"].(string)
			if title == "" {
				log.Fatal("name key not exist in problem.yaml")
			}
			slug = strings.Split(relativePath, "/")[1]
			log.Println("title: ", title)
			log.Println("slug: ", slug)
		}
		if filepath.Base(relativePath) == "problem.md" {
			content, err := os.ReadFile(path)
			if err != nil {
				log.Println(err)
			}
			description := string(content)
			println("description: ", description)
			mapper.CreateProblem(db, model.Problem{
				Slug:        slug,
				Title:       title,
				Description: &description,
				Tags: []*model.AlgorithmTag{
					{Name: "to-be-add"},
				},
			})
		}

		_, minioErr := minioClient.FPutObject(ctx, bucketName,
			relativePath,
			path,
			minio.PutObjectOptions{})
		if minioErr != nil {
			log.Fatalln(minioErr)
		}
		return err
	})

	log.Println("Read Problem Success!")
}
