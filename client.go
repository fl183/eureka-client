package eureka_client

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Client eureka客户端
type Client struct {
	// for monitor system signal
	signalChan chan os.Signal
	mutex      sync.RWMutex
	Running    bool
	Config     *Config
	// eureka服务中注册的应用
	Applications *Applications
	Registered bool
}

func cGetZone(zones []string) string {
	rand.Seed(time.Now().UnixNano())

	var randNum int
	for {
		if randNum = rand.Intn(100); randNum == 0 {
			continue
		}
		log.Println(randNum)
		break
	}

	var zone = zones[randNum%len(zones)]
	log.Println("nums", len(zones), randNum%len(zones))
	log.Println(zones)
	log.Println(zone)
	return zone
}

// Start 启动时注册客户端，并后台刷新服务列表，以及心跳
func (c *Client) Start() {
	c.mutex.Lock()
	c.Running = true
	c.Registered = false
	c.mutex.Unlock()

	c.Config.DefaultZone = cGetZone(c.Config.Zones)

	// 监听退出信号，自动删除注册信息
	go c.handleSignal()

	var cRefresh = make(chan int)
	var cHeartbeat = make(chan int)
	var tryCount = 0
	var serverNum = len(c.Config.Zones)

	for {
		if c.Registered == false {
			// 注册
			c.Registered = true
			tryCount++
			if err := c.doRegister(); err != nil {
				log.Println(err.Error())
				c.Registered = false
				time.Sleep(time.Duration(1)*time.Second)
				if err := c.doRegister(); err != nil {
					log.Println(err.Error())
					c.Registered = false
					time.Sleep(time.Duration(1)*time.Second)
					if err := c.doRegister(); err != nil {
						log.Println(err.Error())
						c.Registered = false
						time.Sleep(time.Duration(1)*time.Second)
						if tryCount < serverNum {
							for i:=0; i<len(c.Config.Zones); i++ {
								if c.Config.Zones[i] == c.Config.DefaultZone {
									c.Config.Zones = append(c.Config.Zones[:i], c.Config.Zones[i+1:]...)
									c.Config.DefaultZone = cGetZone(c.Config.Zones)
								}
							}
							continue
						}
						log.Println("Eureka register failed, Exit...")
						os.Exit(5)
					}
				}
			}
			log.Println("Register application instance successful")
		}

		// 刷新服务列表
		go c.refresh(cRefresh, cHeartbeat)
		// 心跳
		go c.heartbeat(cHeartbeat, cRefresh)

		select {
		case cR := <-cRefresh:
			for i:=0; i<len(c.Config.Zones); i++ {
				if c.Config.Zones[i] == c.Config.DefaultZone {
					c.Config.Zones = append(c.Config.Zones[:i], c.Config.Zones[i+1:]...)
					c.Config.DefaultZone = cGetZone(c.Config.Zones)
				}
			}
			cRefresh <- cR
			time.Sleep(time.Duration(1)*time.Second)
		case cH := <-cHeartbeat:
			for i:=0; i<len(c.Config.Zones); i++ {
				if c.Config.Zones[i] == c.Config.DefaultZone {
					c.Config.Zones = append(c.Config.Zones[:i], c.Config.Zones[i+1:]...)
					c.Config.DefaultZone = cGetZone(c.Config.Zones)
				}
			}
			cHeartbeat <- cH
			time.Sleep(time.Duration(1)*time.Second)
			//default:
		//	client.UnRegister()
		//	os.Exit(1)
		}
	}

}

// refresh 刷新服务列表
func (c *Client) refresh(c1 chan int, c2 chan int) {
	for {
		select {
		case t2 := <-c2:
			c2 <- t2
			break
		default:
			sleep := time.Duration(c.Config.RegistryFetchIntervalSeconds)
			time.Sleep(sleep * time.Second)
		}
		if c.Running {
			if err := c.doRefresh(); err != nil {
				log.Println(err)
				time.Sleep(time.Duration(3)*time.Second)
				if err := c.doRefresh(); err != nil {
					log.Println(err)
					time.Sleep(time.Duration(3)*time.Second)
					if err := c.doRefresh(); err != nil {
						log.Println(err)
						c1 <- 1
						break
					} else {
						log.Println("Refresh application instance successful")
					}
				} else {
					log.Println("Refresh application instance successful")
				}
			} else {
				log.Println("Refresh application instance successful")
			}
		} else {
			break
		}
	}
}

// heartbeat 心跳
func (c *Client) heartbeat(c1 chan int, c2 chan int) {
	for {
		select {
		case t2 := <-c2:
			c2 <- t2
			break
		default:
			sleep := time.Duration(c.Config.RenewalIntervalInSecs)
			time.Sleep(sleep * time.Second)
		}
		if c.Running {
			if err := c.doHeartbeat(); err != nil {
				log.Println(err)
				time.Sleep(time.Duration(3)*time.Second)
				if err := c.doHeartbeat(); err != nil {
					log.Println(err)
					time.Sleep(time.Duration(3)*time.Second)
					if err := c.doHeartbeat(); err != nil {
						log.Println(err)
						c1 <- 1
						break
					} else {
						log.Println("Heartbeat application instance successful")
					}
				} else {
					log.Println("Heartbeat application instance successful")
				}
			} else {
				log.Println("Heartbeat application instance successful")
			}
		} else {
			break
		}
	}
}

func (c *Client) doRegister() error {
	instance := c.Config.instance
	return Register(c.Config.DefaultZone, c.Config.App, instance)
}

func (c *Client) doUnRegister() error {
	instance := c.Config.instance
	return UnRegister(c.Config.DefaultZone, instance.App, instance.InstanceID)
}

func (c *Client) UnRegister() {
	instance := c.Config.instance
	UnRegister(c.Config.DefaultZone, instance.App, instance.InstanceID)
}

func (c *Client) doHeartbeat() error {
	instance := c.Config.instance
	return Heartbeat(c.Config.DefaultZone, instance.App, instance.InstanceID)
}

func (c *Client) doRefresh() error {
	// todo If the delta is disabled or if it is the first time, get all applications

	// get all applications
	applications, err := Refresh(c.Config.DefaultZone)
	if err != nil {
		return err
	}

	// set applications
	c.mutex.Lock()
	c.Applications = applications
	c.mutex.Unlock()
	return nil
}

// handleSignal 监听退出信号，删除注册的实例
func (c *Client) handleSignal() {
	if c.signalChan == nil {
		c.signalChan = make(chan os.Signal)
	}
	signal.Notify(c.signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	for {
		switch <-c.signalChan {
		case syscall.SIGINT:
			fallthrough
		case syscall.SIGKILL:
			fallthrough
		case syscall.SIGTERM:
			log.Println("Receive exit signal, client instance going to de-egister")
			err := c.doUnRegister()
			if err != nil {
				log.Println(err.Error())
			} else {
				log.Println("UnRegister application instance successful")
			}
			c.Running = false
			//os.Exit(0)
		}
	}
}

// NewClient 创建客户端
func NewClient(config *Config) *Client {
	defaultConfig(config)
	config.instance = NewInstance(getLocalIP(), config)
	return &Client{Config: config}
}

func defaultConfig(config *Config) {
	if config.DefaultZone == "" {
		config.DefaultZone = "http://localhost:8761/eureka/"
	}
	if config.RenewalIntervalInSecs == 0 {
		config.RenewalIntervalInSecs = 30
	}
	if config.RegistryFetchIntervalSeconds == 0 {
		config.RegistryFetchIntervalSeconds = 15
	}
	if config.DurationInSecs == 0 {
		config.DurationInSecs = 90
	}
	if config.App == "" {
		config.App = "server"
	} else {
		config.App = strings.ToLower(config.App)
	}
	if config.Port == 0 {
		config.Port = 80
	}
}
