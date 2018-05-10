package logs

import (
	"strconv"
	"gAPIManagement/api/utils"
)

var WorkQueue = make(chan LogWorkRequest, 1000)
var LogWorkQueue chan chan LogWorkRequest

func StartDispatcher(nworkers int) {
	// First, initialize the channel we are going to but the workers' work channels into.
	LogWorkQueue = make(chan chan LogWorkRequest, nworkers)

	// Now, create all of our workers.
	for i := 0; i < nworkers; i++ {
		utils.LogMessage("Starting worker - " + strconv.Itoa(i+1))
		worker := NewWorker(i+1, LogWorkQueue)
		worker.Start()
	}

	go func() {
		for {
			select {
			case work := <-WorkQueue:
				go func() {
					worker := <-LogWorkQueue

					// utils.LogMessage("Dispatching work request")
					worker <- work
				}()
			}
		}
	}()
}
