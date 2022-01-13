package test

import (
	"encoding/json"
	"testing"

	client "github.com/influxdata/influxdb1-client/v2"
)

func TestInfluxQuery(t *testing.T) {
	// Make client
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://localhost:8086",
	})
	if err != nil {
		t.Error("Error creating InfluxDB Client: ", err.Error())
	}
	defer c.Close()

	q := `SELECT round(difference(mean("value"))) FROM "cpu_usage_total"
	WHERE ("container_name" = '/') AND time >= now() - 1h and time <= now()
	GROUP BY time(1m) fill(null);`
	/*
		q := client.NewQueryWithParameters("SELECT $fn($value) FROM $m", "cadvisor", "s", client.Params{
			"fn":    client.Identifier("count"),
			"value": client.Identifier("value"),
			"m":     client.Identifier("shapes"),
		})
	*/
	response, err := c.Query(client.NewQuery(q, "cadvisor", "s"))
	if err != nil {
		t.Error(err)
	} else if response.Error() != nil {
		t.Log(response.Error())
	}

	for _, r := range response.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				ts, _ := v[0].(json.Number).Int64()
				usage, _ := v[1].(json.Number).Float64()
				usage /= (60 * 1e9)
				t.Logf("%v %v", ts, usage)
			}
		}
	}
}
