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
	topic         = "CAR_NUMBER_GO"
	config        *sarama.Config
	kafkaServer   = flag.String("kafkaServer", "47.105.36.188:9092", "")
)

func init() {
	flag.Parse()
	pool = newPool(*redisServer, *redisPassword)

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
	conn := pool.Get()
	defer conn.Close()
	conn.Send("MULTI")
	conn.Send("GET", "counter")
	conn.Send("DECR", "counter")
	num, err := redis.Int64s(conn.Do("EXEC"))
	if err != nil {
		fmt.Println(err)
		return
	}
	if num[1] >= 0 {
		messages := "go_购买成功，还剩下" + fmt.Sprintf("%d", num[1]) + "个"
		fmt.Println(messages)
		this.Data["json"] = map[string]interface{}{"messages": messages}
		//设置kafka配置
		config := sarama.NewConfig()
		//是否等待成功和失败后的响应,只有上面的RequireAcks设置不是NoReponse这里才有用.
		config.Producer.Return.Successes = true
		config.Producer.Return.Errors = true
		//使用配置,新建一个异步生产者
		producer, e := sarama.NewAsyncProducer([]string{*kafkaServer}, config)
		if e != nil {
			panic(e)
		}
		defer producer.AsyncClose()
		//发送的消息,主题,key
		msg := &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.StringEncoder(messages),
		}
		producer.Input() <- msg
		fmt.Println("开始发送")
		//循环判断哪个通道发送过来数据.
		select {
		case suc := <-producer.Successes():
			fmt.Println("发送成功 offset: ", suc.Offset, "timestamp: ", suc.Timestamp.String(), "partitions: ", suc.Partition)
		case fail := <-producer.Errors():
			fmt.Println("发送失败 err: ", fail.Err)
		}
	} else {
		fmt.Print("抢完了")
		this.Data["json"] = map[string]interface{}{"messages": "抢完了"}
	}
	this.ServeJSON()

}

func (this *GetCountController) Get() {
	conn := pool.Get()
	defer conn.Close()
	r, _ := redis.Int(conn.Do("GET", "counter"))
	this.Data["json"] = map[string]interface{}{"num": r}
	this.ServeJSON()
}

//初始化一个pool
func newPool(server, password string) *redis.Pool {
	return &redis.Pool{ //实例化一个连接池
		MaxIdle:     3, //最初的连接数量
		MaxActive:   0, //连接池最大连接数量,不确定可以用0（0表示自动定义），按需分配
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
