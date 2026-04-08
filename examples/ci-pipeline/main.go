// ci-pipeline demonstrates the typical CI/CD workflow:
// update the Docker image tag on an existing service and wait for it to roll out.
//
// Environment variables required:
//
//	CLOUD_TOKEN     — project token (sk_proj_*)
//	CLOUD_WORKSPACE — workspace slug
//	CLOUD_PROJECT   — project slug
//	CLOUD_ENV       — environment slug (e.g. "production")
//	SERVICE_SLUG    — service slug to update
//	IMAGE           — Docker image (e.g. "ghcr.io/my-org/api")
//	IMAGE_TAG       — new image tag (e.g. the git SHA or semver)
//
// Usage:
//
//	IMAGE_TAG=$(git rev-parse --short HEAD) go run ./examples/ci-pipeline
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	cloud "github.com/simplifyd-systems/cloud-go-sdk"
)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("environment variable %s is required", key)
	}
	return v
}

func main() {
	token     := mustEnv("CLOUD_TOKEN")
	workspace := mustEnv("CLOUD_WORKSPACE")
	project   := mustEnv("CLOUD_PROJECT")
	env       := mustEnv("CLOUD_ENV")
	svcSlug   := mustEnv("SERVICE_SLUG")
	image     := mustEnv("IMAGE")
	tag       := mustEnv("IMAGE_TAG")

	client := cloud.NewClient(cloud.WithToken(token))
	ctx := context.Background()

	svcs := client.Workspace(workspace).Project(project).Env(env).Services()

	fmt.Printf("Deploying %s:%s to %s/%s/%s ...\n", image, tag, workspace, project, env)

	dep, err := svcs.DeployImage(ctx, svcSlug, image, tag)
	if err != nil {
		log.Fatalf("deploy image: %v", err)
	}
	fmt.Printf("Deployment %s started\n", dep.Slug)

	// Stream logs while waiting.
	logCtx, cancelLog := context.WithCancel(ctx)
	go func() {
		_ = svcs.StreamLogs(logCtx, svcSlug, dep.Slug, func(line string) {
			fmt.Println(" >", line)
		})
	}()

	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	final, err := svcs.WaitForDeployment(waitCtx, svcSlug, dep.Slug, 5*time.Second)
	cancelLog()
	if err != nil {
		log.Fatalf("waiting for deployment: %v", err)
	}

	switch final.Status {
	case cloud.DeploymentStatusRunning:
		fmt.Printf("Deployment successful — %s:%s is live\n", image, tag)
	default:
		fmt.Printf("Deployment ended with status: %s\n", final.Status)
		os.Exit(1)
	}
}
