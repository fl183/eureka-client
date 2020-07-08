package eureka_client

import (
	"fmt"
	"net/url"

	"github.com/xuanbo/requests"
)

// 与eureka服务端rest交互
// https://github.com/Netflix/eureka/wiki/Eureka-REST-operations

func getZone(zones string) string {
	//var eureServerkaList = strings.Split(zones, ",")
	//rand.Seed(time.Now().UnixNano())
	//
	//var zone = eureServerkaList[rand.Intn(100)%len(eureServerkaList)]
	return zones
}

//func getZoneRetry(zones string, current string) string {
//	var finalZone []string
//	for bad := range current {
//		var temps = strings.Split(zones)
//
//		for temp := range temps {
//			if temp == bad {
//				continue
//			} else {
//				finalZone = append(ret, temp)
//			}
//		}
//	}
//	rand.Seed(time.Now().UnixNano())
//	var zone = finalZone[rand.Intn(100)%len(finalZone)]
//	return zone
//}

// Register 注册实例
// POST /eureka/v2/apps/appID
func Register(zone, app string, instance *Instance) error {
	// Instance 服务实例
	type InstanceInfo struct {
		Instance *Instance `json:"instance"`
	}
	var info = &InstanceInfo{
		Instance: instance,
	}

	url := getZone(zone) + "apps/" + app

	// status: http.StatusNoContent
	result := requests.Post(url).Json(info).Send().Status2xx()
	if result.Err != nil {
		return fmt.Errorf("Register application instance failed, error: %s", result.Err)
	}
	return nil
}

// UnRegister 删除实例
// DELETE /eureka/v2/apps/appID/instanceID
func UnRegister(zone, app, instanceID string) error {
	url := getZone(zone) + "apps/" + app + "/" + instanceID
	// status: http.StatusNoContent
	result := requests.Delete(url).Send().StatusOk()
	if result.Err != nil {
		return fmt.Errorf("UnRegister application instance failed, error: %s", result.Err)
	}
	return nil
}

// Refresh 查询所有服务实例
// GET /eureka/v2/apps
func Refresh(zone string) (*Applications, error) {
	type Result struct {
		Applications *Applications `json:"applications"`
	}
	apps := new(Applications)
	res := &Result{
		Applications: apps,
	}
	url := getZone(zone) + "apps"
	err := requests.Get(url).Header("Accept", " application/json").Send().StatusOk().Json(res)
	if err != nil {
		return nil, fmt.Errorf("Refresh failed, error: %s", err)
	}
	return apps, nil
}

// Heartbeat 发送心跳
// PUT /eureka/v2/apps/appID/instanceID
func Heartbeat(zone, app, instanceID string) error {
	u := getZone(zone) + "apps/" + app + "/" + instanceID
	params := url.Values{
		"status": {"UP"},
	}
	result := requests.Put(u).Params(params).Send().StatusOk()
	if result.Err != nil {
		return fmt.Errorf("Heartbeat failed, error: %s", result.Err)
	}
	return nil
}
