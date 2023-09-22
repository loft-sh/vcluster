package strvals

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/assert"
)

func TestSetStringFlag(t *testing.T) {
	rawConfig := getActualRawConfig()
	s := "deployments.dev.helm.values.containers[0]="
	err := ParseIntoString(s, rawConfig)
	if err != nil {
		fmt.Println(errors.Wrap(err, "parsing --set-string flag"))
		log.Fatal(err)
	}
	b, err := json.Marshal(rawConfig)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("output : " + string(b))
	assert.DeepEqual(t, rawConfig, getExpectedRawConfigForSetString())
}

func TestSetFlag(t *testing.T) {
	s := "deployments.dev.helm.values.containers[1].image="
	rawConfig := getActualRawConfig()
	err := ParseInto(s, rawConfig)
	if err != nil {
		fmt.Println(errors.Wrap(err, "parsing --set flag"))
		log.Fatal(err)
	}
	b, err := json.Marshal(rawConfig)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("output : " + string(b))
	assert.DeepEqual(t, rawConfig, getExpectedRawConfigForSet())
}

func getExpectedRawConfigForSet() map[string]interface{} {
	jsonStr := "{\"deployments\":{\"dev\":{\"helm\":{\"values\":{\"containers\":[{\"image\":\"alpine\"},{\"image\":null}]}}}},\"name\":\"run-pipelines-demo\",\"pipelines\":{\"deploy\":\"create_deployments --all\",\"dev\":\"run_pipelines deploy --set deployments.dev.helm.values.containers[0].image=nginx --set-string deployments.dev.helm.values.containers[0].name=mynginx\"},\"version\":\"v2beta1\"}"
	rawConfig := map[string]interface{}{}
	err := json.Unmarshal([]byte(jsonStr), &rawConfig)
	if err != nil {
		fmt.Println(err)
	}
	return rawConfig
}

func getExpectedRawConfigForSetString() map[string]interface{} {
	jsonStr := "{\"deployments\":{\"dev\":{\"helm\":{\"values\":{\"containers\":[null,{\"image\":\"ns\"}]}}}},\"name\":\"run-pipelines-demo\",\"pipelines\":{\"deploy\":\"create_deployments --all\",\"dev\":\"run_pipelines deploy --set deployments.dev.helm.values.containers[0].image=nginx --set-string deployments.dev.helm.values.containers[0].name=mynginx\"},\"version\":\"v2beta1\"}"
	rawConfig := map[string]interface{}{}
	err := json.Unmarshal([]byte(jsonStr), &rawConfig)
	if err != nil {
		fmt.Println(err)
	}
	return rawConfig
}

func getActualRawConfig() map[string]interface{} {
	jsonStr := "{\n  \"deployments\": {\n    \"dev\": { \"helm\": { \"values\": { \"containers\": [{ \"image\": \"alpine\" },{ \"image\": \"ns\" }] } } }\n  },\n  \"name\": \"run-pipelines-demo\",\n  \"pipelines\": {\n    \"deploy\": \"create_deployments --all\",\n    \"dev\": \"run_pipelines deploy --set deployments.dev.helm.values.containers[0].image=nginx --set-string deployments.dev.helm.values.containers[0].name=mynginx\"\n  },\n  \"version\": \"v2beta1\"\n}\n"
	rawConfig := map[string]interface{}{}
	err := json.Unmarshal([]byte(jsonStr), &rawConfig)
	if err != nil {
		fmt.Println(err)
	}
	return rawConfig
}
