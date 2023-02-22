package main

import (
	"agent/config"
	"agent/util"
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron/v3"
)

var (
	localCache = cache.New(5*time.Minute, 10*time.Minute)
	ctx        = context.Background()
	cfg        = config.LoadConfig()
)

type Service struct {
	Name    string    `yaml:"name" json:"name"`
	CN      string    `yaml:"cn" json:"cn"`
	Health  bool      `yaml:"health" json:"health"`
	CreatAt time.Time `yaml:"creatAt" json:"creatAt"`
	StartAt time.Time `yaml:"startAt" json:"startAt"`
	ExitAt  time.Time `yaml:"exitAt" json:"exitAt"`
}

type CacheEntry struct {
	Service   Service
	Container types.ContainerJSON
}

func main() {
	loadContainer()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	c := cron.New(cron.WithSeconds())
	_, err = c.AddFunc("*/5 * * * * *", func() {
		check()
	})
	_, err = c.AddFunc("* */5 * * * *", func() {
		loadContainer()
	})
	if err != nil {
		return
	}
	c.Start()

	r := gin.Default()
	r.GET("/*service", wrapper(handlerServiceRequest))
	r.Run(":9090")
}

func loadContainer() {
	output, _ := util.Cmd("docker ps -a --format=\"{{.ID}}\"")
	lines := strings.Split(output, "\n")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	for _, c := range lines {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}

		containerJSON, err := cli.ContainerInspect(ctx, c)
		if err != nil {
			log.Fatal(err)
		}
		configFiles := containerJSON.Config.Labels["com.docker.compose.project.config_files"]
		if configFiles == "" || !strings.HasPrefix(configFiles, "/platform/") {
			continue
		}
		t := strings.Split(configFiles, "/")
		name := t[len(t)-1]
		name = strings.TrimSuffix(name, ".yaml")
		name = strings.TrimSuffix(name, ".yml")

		timePattern := "2006-01-02T15:04:05.999999999Z"
		creatAt, err := time.Parse(timePattern, containerJSON.Created)
		exitAt := time.Time{}
		if !containerJSON.State.Running {
			exitAt, err = time.Parse(timePattern, containerJSON.State.FinishedAt)
		}

		entry, ok := localCache.Get(name)
		if !ok {
			localCache.Set(name, CacheEntry{
				Service: Service{
					Name:    name,
					CN:      "",
					Health:  false,
					CreatAt: creatAt,
					StartAt: time.Time{},
					ExitAt:  exitAt,
				},
				Container: containerJSON,
			}, cache.NoExpiration)
		} else {
			localCache.Set(name, CacheEntry{
				Service:   entry.(CacheEntry).Service,
				Container: containerJSON,
			}, cache.NoExpiration)
		}
	}
}

func check() {
	for n, s := range cfg.Services {
		e, ok := localCache.Get(n)
		if !ok {
			fmt.Println(n)
		}
		cacheEntry := e.(CacheEntry)
		cacheEntry.Service.CN = s.CN
		switch s.HealthCheckMode {
		case "":
			cacheEntry.Service.Health = true
			cacheEntry.Service.StartAt = time.Now()
			break
		case "docker-command":
			_, exitCode := util.DockerCmd(cacheEntry.Container.Name, s.HealthCheckCmd)
			if exitCode == 0 {
				cacheEntry.Service.Health = true
				cacheEntry.Service.ExitAt = time.Time{}
				if cacheEntry.Service.StartAt.IsZero() {
					cacheEntry.Service.StartAt = time.Now()
				}
			} else {
				cacheEntry.Service.Health = false
				cacheEntry.Service.StartAt = time.Time{}
				if !cacheEntry.Service.ExitAt.IsZero() {
					cacheEntry.Service.ExitAt = time.Now()
				}
			}
			break
		case "http":
			for port := range cacheEntry.Container.NetworkSettings.Ports {
				if util.HttpHealthCheck(port.Int(), s.HealthCheckEndpoint) {
					cacheEntry.Service.Health = true
					cacheEntry.Service.ExitAt = time.Time{}
					if cacheEntry.Service.StartAt.IsZero() {
						cacheEntry.Service.StartAt = time.Now()
					}
					goto LoopEnd
				}
			}
			cacheEntry.Service.Health = false
			cacheEntry.Service.StartAt = time.Time{}
			if !cacheEntry.Service.ExitAt.IsZero() {
				cacheEntry.Service.ExitAt = time.Now()
			}
		LoopEnd:
			break
		case "nacos":
			if util.NacosHealthCheck(s.Name, s.NacosNamespace, s.NacosUsername, s.NacosPassword) {
				cacheEntry.Service.Health = true
				cacheEntry.Service.ExitAt = time.Time{}
				if cacheEntry.Service.StartAt.IsZero() {
					cacheEntry.Service.StartAt = time.Now()
				}
			} else {
				cacheEntry.Service.Health = false
				cacheEntry.Service.StartAt = time.Time{}
				if !cacheEntry.Service.ExitAt.IsZero() {
					cacheEntry.Service.ExitAt = time.Now()
				}
			}
			break
		default:
			break
		}
		localCache.Set(n, cacheEntry, cache.NoExpiration)
	}
}

type HandlerFunc func(c *gin.Context) error

func wrapper(handler HandlerFunc) func(c *gin.Context) {
	return func(c *gin.Context) {
		err := handler(c)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
	}
}

func handlerServiceRequest(c *gin.Context) error {
	service := c.Param("service")
	service = service[1:]
	if service != "" {
		s, ok := localCache.Get(service)
		if ok {
			c.JSON(200, s.(CacheEntry).Service)
		} else {
			c.Status(200)
		}
		return nil
	}
	var services []Service
	for _, item := range localCache.Items() {
		services = append(services, item.Object.(CacheEntry).Service)
	}
	c.JSON(200, services)
	return nil
}
