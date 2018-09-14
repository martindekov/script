package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/openfaas/faas/gateway/requests"
)

var newLabels = map[string]string{
	"com.openfaas.cloud.git-cloud":      "Git-Cloud",
	"com.openfaas.cloud.git-deploytime": "Git-DeployTime",
	"com.openfaas.cloud.git-owner":      "Git-Owner",
	"com.openfaas.cloud.git-repo":       "Git-Repo",
	"com.openfaas.cloud.git-SHA":        "Git-SHA",
}

// Handle a serverless request
func Handle(req []byte) string {
	gatewayURL := os.Getenv("url")
	httpReq, reqErr := http.NewRequest(http.MethodGet, gatewayURL+"/system/functions", nil)
	if reqErr != nil {
		fmt.Printf("The request error is not nil")
	}
	authErr := AddBasicAuth(httpReq)
	if authErr != nil {
		fmt.Printf("AuthErr is not nil")
		return fmt.Sprintf("Got err: `%v`", authErr.Error())
	}
	var Requests []requests.CreateFunctionRequest
	c := http.Client{
		Timeout: 3 * time.Second,
	}
	httpRes, resErr := c.Do(httpReq)
	fmt.Printf("Reached response")

	if resErr != nil {
		fmt.Printf("Response is not nil")
		return fmt.Sprintf("Got err: `%v`", resErr.Error())
	}

	body, _ := ioutil.ReadAll(httpRes.Body)
	err := json.Unmarshal(body, &Requests)
	if err != nil {
		fmt.Printf("EWrror occured")
	}
	httpRes.Body.Close()
	for _, request := range Requests {
		labels := *request.Labels

		for k, v := range newLabels {
			if labels[v] != "" {
				labels[k] = labels[v]
				delete(labels, v)
			}
		}
		//bs, err := json.Marshal(Requests)
		bs, err := json.Marshal(request)
		if err != nil {
			fmt.Println(err.Error())
		}

		//fmt.Println("requests", string(bs))

		buf := bytes.NewBuffer(bs)

		httpReq, reqErr = http.NewRequest(http.MethodPut, gatewayURL+"/system/functions", buf)
		if reqErr != nil {
			fmt.Printf("The request error is not nil")
		}
		authErr = AddBasicAuth(httpReq)
		httpReq.Header.Set("Content-Type", "application/json")
		if authErr != nil {
			fmt.Printf("AuthErr is not nil")
			return fmt.Sprintf("Got err: `%v`", authErr.Error())
		}

		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		res, err := client.Do(httpReq)
		if res.StatusCode != 200 {
			fmt.Printf("`%v`", res.StatusCode)
		}
		if err != nil {
			fmt.Printf(err.Error())
		}
		fmt.Printf("Status: %v\n", res.Status)
		fmt.Printf("\n")
		res.Body.Close()
	}

	return "wat"
}

// BasicAuthCredentials for credentials

type BasicAuthCredentials struct {
	User     string
	Password string
}

type ReadBasicAuth interface {
	Read() (error, *BasicAuthCredentials)
}

type ReadBasicAuthFromDisk struct {
	SecretMountPath string
}

func (r *ReadBasicAuthFromDisk) Read() (*BasicAuthCredentials, error) {
	var credentials *BasicAuthCredentials

	if len(r.SecretMountPath) == 0 {
		return nil, fmt.Errorf("invalid SecretMountPath specified for reading secrets")
	}

	userPath := path.Join(r.SecretMountPath, "basic-auth-user")
	user, userErr := ioutil.ReadFile(userPath)
	if userErr != nil {
		return nil, fmt.Errorf("unable to load %s", userPath)
	}

	userPassword := path.Join(r.SecretMountPath, "basic-auth-password")
	password, passErr := ioutil.ReadFile(userPassword)
	if passErr != nil {
		return nil, fmt.Errorf("Unable to load %s", userPassword)
	}

	credentials = &BasicAuthCredentials{
		User:     strings.TrimSpace(string(user)),
		Password: strings.TrimSpace(string(password)),
	}

	return credentials, nil
}

func AddBasicAuth(req *http.Request) error {
	if len(os.Getenv("basic_auth")) > 0 && os.Getenv("basic_auth") == "true" {

		reader := ReadBasicAuthFromDisk{}

		if len(os.Getenv("secret_mount_path")) > 0 {
			reader.SecretMountPath = os.Getenv("secret_mount_path")
		}

		credentials, err := reader.Read()

		if err != nil {
			return fmt.Errorf("error with AddBasicAuth %s", err.Error())
		}

		req.SetBasicAuth(credentials.User, credentials.Password)
	}
	return nil
}
