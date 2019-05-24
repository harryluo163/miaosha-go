package controllers

import (
	"flag"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/astaxie/beego"
	"github.com/garyburd/redigo/redis"
	"time"
)
//声明一些全局变量
var (
	pool          *redis.Pool
	redisServer   = flag.String("redisServer", "47.105.36.188:6379", "")
	redisPassword = flag.String("redisPassword", "test123", "")
)
//初始化一个pool
func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("AUTH", password); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

type MainController struct {
	beego.Controller
}
type SeckillController struct {
	beego.Controller
}
type GetCountController struct {
	beego.Controller
}
func (c *MainController) Get() {
	c.TplName = "index.html"
}
func (this *SeckillController) Post() {
	flag.Parse()
	pool = newPool(*redisServer, *redisPassword)
	conn := pool.Get()
	defer conn.Close()
	conn.Send("MULTI")
	conn.Send("DECR", "counter")
	num, err := conn.Do("EXEC")
	if err ==nil {
		fmt.Print("购买成功，还剩下"+fmt.Sprintf("%d", num)+"个")
		this.Data["json"] = map[string]interface{}{"messages": "购买成功，还剩下"+fmt.Sprintf("%d", num)+"个"}
		//设置配置
		config := sarama.NewConfig()


		//是否等待成功和失败后的响应,只有上面的RequireAcks设置不是NoReponse这里才有用.
		config.Producer.Return.Successes = true
		config.Producer.Return.Errors = true


		//使用配置,新建一个异步生产者
		producer, e := sarama.NewAsyncProducer([]string{"47.105.36.188:9092"}, config)
		if e != nil {
			panic(e)
		}
		defer producer.AsyncClose()

		//发送的消息,主题,key
		msg := &sarama.ProducerMessage{
			Topic: "CAR_NUMBER",
			Key:   sarama.StringEncoder("test"),
		}

		var value string

		value = "购买成功，还剩下"+fmt.Sprintf("%d", num)+"个"
		//将字符串转化为字节数组
		msg.Value = sarama.ByteEncoder(value)
		//使用通道发送
		producer.Input() <- msg
		//循环判断哪个通道发送过来数据.
		select {
		case suc := <-producer.Successes():
			fmt.Println("offset: ", suc.Offset, "timestamp: ", suc.Timestamp.String(), "partitions: ", suc.Partition)
		case fail := <-producer.Errors():
			fmt.Println("err: ", fail.Err)
		}



	}else{
		fmt.Print("抢完了")
		this.Data["json"] = map[string]interface{}{"messages": "抢完了"}
	}
	this.ServeJSON()

}

func (this *GetCountController) Get() {
	flag.Parse()
	pool = newPool(*redisServer, *redisPassword)
	conn := pool.Get()
	defer conn.Close()
	r, _ :=conn.Do("GET", "counter")
	this.Data["json"] = map[string]interface{}{"num": r}
	this.ServeJSON()
}