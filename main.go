package main

import (
	"fmt"
	"log"
	"os"
	"github.com/arijiiiitttt/dinoDB/api"
	"github.com/arijiiiitttt/dinoDB/engine"
	"github.com/arijiiiitttt/dinoDB/repl"
	_ "image/jpeg"
	_ "image/png"

	"github.com/qeesung/image2ascii/convert"
)

func main() {
	dataDir := "./dinoDB"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	db, err := engine.Open(dataDir)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close() 

	imageFilename := "image/cmd.png"

	opt := convert.DefaultOptions
	opt.FixedWidth = 94
	opt.FixedHeight = 30
	opt.Colored = true

	converter := convert.NewImageConverter()

	fmt.Println(converter.ImageFile2ASCIIString(imageFilename, &opt))

	fmt.Printf("  HTTP REST API  : http://localhost:8080\n")
	server := api.NewServer(db, ":8080")
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	r := repl.New(db)
	r.Run()
}
