package httpclient

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const hTPClientTimeout = 10 * time.Second
