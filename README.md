# dockerdyn

Dynamic labeling system for docker containers.

### Install
```go get github.com/saromanov/dockerdyn```

### Usage
```go
package main

import (
  "github.com/saromanov/dockerdyn"
)

func main() {
	dock := dockerdyn.New()
	dock.AddHandlerInspect("Name", func(i interface{}) string {
		if i.(string) == "redis" || i.(string) == "mongo"{
			return "db"
		}

		return "unknown"
	})
	dock.Start()
}
