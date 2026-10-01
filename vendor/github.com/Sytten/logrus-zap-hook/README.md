# Zap Hook for Logrus <img src="http://i.imgur.com/hTeVwmJ.png" width="40" height="40" alt=":walrus:" class="emoji" title=":walrus:"/>

Use this hook to send logs from [logrus](https://github.com/sirupsen/logrus) to [zap](https://github.com/uber-go/zap).
All levels are sent by default.

## Usage

```go
package main

import (
    "io/ioutil"
    
    zaphook "github.com/Sytten/logrus-zap-hook"
    "github.com/sirupsen/logrus"
    "go.uber.org/zap"
)

func main() {
    log := logrus.New()
    log.ReportCaller = true // So Zap reports the right caller
    log.SetOutput(ioutil.Discard) // Prevent logrus from writing its logs
    
    logger, _ := zap.NewDevelopment()
    hook, _ := zaphook.NewZapHook(logger)
    
    log.Hooks.Add(hook)
}
```

