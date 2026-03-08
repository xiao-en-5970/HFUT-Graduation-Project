package main

import (
	"log"

	_ "github.com/xiao-en-5970/HFUT-Graduation-Project/package/schools/hfut" // 注册 HFUT 学校登录
	"github.com/xiao-en-5970/HFUT-Graduation-Project/bootstrap"
)

func main() {
	// Boot the application (initializes all components and starts the server)
	if err := bootstrap.Boot(); err != nil {
		log.Fatalf("Failed to bootstrap application: %v", err)
	}
}
