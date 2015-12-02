package dockerdyn

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/redis.v3"
	"log"
	"reflect"
	"time"
)

// NOTE: After append 'docker set', must be reimplementing core paer

type (
	handler func(interface{}) string
)

type Dockerdyn struct {
	Timeout     time.Duration
	labelsinsp  map[string]handler
	labelsstat  map[string]handler
	labels      map[string][]string
	redisclient *redis.Client
}

func New() *Dockerdyn {
	dd := new(Dockerdyn)
	dd.labelsstat = map[string]handler{}
	dd.labelsinsp = map[string]handler{}
	dd.labels = map[string][]string{}
	dd.redisclient = initRedis()
	dd.Timeout = 5 * time.Second
	return dd
}

func (dd *Dockerdyn) AddHandlerInspect(name string, hand handler) {
	dd.labelsinsp[name] = hand
	dd.labels[name] = []string{}
}

func (dd *Dockerdyn) AddHandlerStat(name string, hand handler) {
	dd.labelsstat[name] = hand
	dd.labels[name] = []string{}
}

func initRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

func (dd *Dockerdyn) Start() {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	imgs, _ := client.ListContainers(docker.ListContainersOptions{All: false})

	// TODO
	for _, img := range imgs {
		go func(id string) {
			statchan := make(chan *docker.Stats)
			done := make(chan bool)
			errstat := client.Stats(docker.StatsOptions{ID: id, Stats: statchan, Stream: true, Done: done})
			if errstat != nil {
				log.Fatal(errstat)
			}
			select {
			case stat := <-statchan:
				fmt.Println(stat)
			default:
				fmt.Println("NO")
			}
		}(img.ID)
	}

	for {
		imgs, err2 := client.ListContainers(docker.ListContainersOptions{All: false})
		if err2 != nil {
			log.Fatal(err2)
		}
		if len(imgs) == 0 {
			fmt.Println("Not found of active containers")
		}

		for _, img := range imgs {
			inspect, err := client.InspectContainer(img.ID)
			if err != nil {
				continue
			}

			for label, hand := range dd.labelsinsp {
				valueof := reflect.ValueOf(inspect)
				value := reflect.Indirect(valueof).FieldByName(label)
				if value.Kind() != reflect.Invalid {
					dd.addID(hand(value.Interface()), img.ID)
				}

			}
		}
		time.Sleep(dd.Timeout)
	}
}

// containsID returns true if target id contains in label map
// and false otherwise
func (dd *Dockerdyn) containsID(label, id string) bool {
	list, _ := dd.labels[label]

	for _, value := range list {
		if value == id {
			return true
		}
	}
	return false
}

func (dd *Dockerdyn) removeID(label, id string) error {
	list, ok := dd.labels[label]
	if !ok {
		return fmt.Errorf("%s in the label %s is not found")
	}

	num := -1
	for i, value := range list {
		if value == id {
			num = i
			break
		}
	}

	if num == -1 {
		return fmt.Errorf("%s in the label %s is not found")
	}

	list = append(list[:num], list[num+1:]...)
	dd.labels[label] = list
	return nil
}

func (dd *Dockerdyn) addID(label, id string) {
	if !dd.containsID(label, id) {
		// Probably label for ID is changing and this ID must be removed from other labels
		for lab, _ := range dd.labels {
			dd.removeID(lab, id)
		}
		dd.labels[label] = append(dd.labels[label], id)
	}
}
