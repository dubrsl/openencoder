package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/alfg/enc/api/data"
	"github.com/alfg/enc/api/types"
	"github.com/gin-gonic/gin"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/xid"
)

var redisPool = &redis.Pool{
	MaxActive: 5,
	MaxIdle:   5,
	Wait:      true,
	Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	},
}

var enqueuer = work.NewEnqueuer("enc", redisPool)

type request struct {
	Profile     string `json:"profile" binding:"required"`
	Source      string `json:"source" binding:"required"`
	Destination string `json:"dest" binding:"required"`
	Delay       string `json:"delay" binding:"required"`
}

type response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type index struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Docs    string `json:"docs"`
	Github  string `json:"github"`
}

func indexHandler(c *gin.Context) {
	resp := index{
		Name:    "enc",
		Version: "0.0.1",
		Docs:    "http://localhost/",
		Github:  "https://github.com/alfg/enc",
	}
	c.JSON(200, resp)
}

func encodeHandler(c *gin.Context) {

	// Decode json.
	var json request
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create Job and push the work to work queue.
	job := types.Job{
		GUID:        xid.New().String(),
		Profile:     json.Profile,
		Source:      json.Source,
		Destination: json.Destination,
	}

	// Send to work queue.
	_, err := enqueuer.Enqueue("encode", work.Q{
		"guid":        job.GUID,
		"profile":     job.Profile,
		"source":      job.Source,
		"destination": job.Destination,
	})
	if err != nil {
		log.Fatal(err)
	}

	created := data.CreateJob(job)
	fmt.Println(created)

	// Create response.
	resp := response{
		Message: "Job created",
		Status:  200,
	}
	c.JSON(http.StatusCreated, resp)
}

func jobsHandler(c *gin.Context) {
	jobs := data.GetJobs()
	c.JSON(http.StatusOK, jobs)
}

func workerQueuesHandler(c *gin.Context) {
	client := work.NewClient("enc", redisPool)

	queues, err := client.Queues()
	if err != nil {
		fmt.Println(err)
	}
	c.JSON(200, queues)
}

func workerPoolsHandler(c *gin.Context) {
	client := work.NewClient("enc", redisPool)

	resp, err := client.WorkerPoolHeartbeats()
	if err != nil {
		fmt.Println(err)
	}
	c.JSON(200, resp)
}

func workerBusyHandler(c *gin.Context) {
	client := work.NewClient("enc", redisPool)

	observations, err := client.WorkerObservations()
	if err != nil {
		fmt.Println(err)
	}

	var busyObservations []*work.WorkerObservation
	for _, ob := range observations {
		if ob.IsBusy {
			busyObservations = append(busyObservations, ob)
		}
	}
	c.JSON(200, busyObservations)
}
