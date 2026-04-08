// deploy-docker demonstrates a complete first-time service deployment:
//   1. Create a Docker service
//   2. Set environment variables
//   3. Add an HTTP ingress port
//   4. Deploy and wait for it to become healthy
//
// Usage:
//
//	CLOUD_TOKEN=sk_proj_... go run ./examples/deploy-docker
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	cloud "github.com/simplifyd-systems/cloud-go-sdk"
)

func main() {
	token := os.Getenv("CLOUD_TOKEN")
	if token == "" {
		log.Fatal("CLOUD_TOKEN environment variable is required")
	}

	// Hard-code your workspace/project/env slugs, or resolve them dynamically.
	const (
		workspace = "my-workspace-slug"
		project   = "my-project-slug"
		env       = "production"
	)

	client := cloud.NewClient(cloud.WithToken(token))
	ctx := context.Background()

	svcs := client.Workspace(workspace).Project(project).Env(env).Services()

	// 1. Create the service.
	fmt.Println("Creating service...")
	svc, err := svcs.Create(ctx, cloud.CreateServiceInput{
		Name: "api-server",
		Type: cloud.ServiceTypeDocker,
		Docker: &cloud.DockerInput{
			Image: "ghcr.io/my-org/api",
			Tag:   "v1.2.3",
		},
		VCPUs:  500,  // 0.5 vCPU
		Memory: 512,  // 512 MiB
	})
	if err != nil {
		log.Fatalf("create service: %v", err)
	}
	fmt.Printf("Created: %s (%s)\n", svc.Name, svc.Slug)

	// 2. Set environment variables.
	fmt.Println("Setting variables...")
	vars := svcs.Variables(svc.Slug)
	if err := vars.BulkSet(ctx, map[string]string{
		"PORT":         "8080",
		"LOG_LEVEL":    "info",
		"DATABASE_URL": "postgres://...",
	}); err != nil {
		log.Fatalf("set variables: %v", err)
	}

	// 3. Add HTTP ingress on port 8080.
	fmt.Println("Adding ingress...")
	port, err := svcs.Ingress(svc.Slug).Add(ctx, cloud.AddIngressInput{
		Protocol: "HTTP",
		Port:     8080,
	})
	if err != nil {
		log.Fatalf("add ingress: %v", err)
	}
	fmt.Printf("Ingress: %s\n", port.VanityFQDN)

	// 4. Deploy and wait.
	fmt.Println("Deploying...")
	dep, err := svcs.Deploy(ctx, svc.Slug)
	if err != nil {
		log.Fatalf("deploy: %v", err)
	}
	fmt.Printf("Deployment started: %s\n", dep.Slug)

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	final, err := svcs.WaitForDeployment(waitCtx, svc.Slug, dep.Slug, 5*time.Second)
	if err != nil {
		log.Fatalf("waiting for deployment: %v", err)
	}

	if final.Status == cloud.DeploymentStatusRunning {
		fmt.Printf("Service is live at https://%s\n", port.VanityFQDN)
	} else {
		fmt.Printf("Deployment ended with status: %s\n", final.Status)
		os.Exit(1)
	}
}
