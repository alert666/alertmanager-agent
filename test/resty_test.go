package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func TestResty(t *testing.T) {
	res, err := resty.New().SetTimeout(time.Second * 2).R().Get("http://172.18.241.234:30420/-/healthy")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("res.Status(): %v\n", res.Status())
	fmt.Printf("res.StatusCode(): %v\n", res.StatusCode())
	fmt.Printf("res: %v\n", res)
}
