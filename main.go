package main

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	_ "miaosha-go/routers"
	"sync"
	"time"
)

var (
	wg sync.WaitGroup
)

func main() {
	//注册Kafka消费者
	orm.RegisterDataBase("default", "mysql", "root:Root!!2018@tcp(47.105.36.188:3306)/seckill?charset=utf8")
	go KafkaConsumer()
	beego.Run()

}

func KafkaConsumer() {
	var (
		offset int64 = 0
		config       = sarama.NewConfig()
		o            = orm.NewOrm()
	)
	//获取最大offset值，因为kafka中每个message在有一个唯一值 就是offset,避免
	o.Raw("select max(offset) as offset from seckill").QueryRow(&offset)
	//接收失败通知
	config.Consumer.Return.Errors = true
	// 根据给定的代理地址和配置创建一个消费者
	consumer, err := sarama.NewConsumer([]string{"47.105.36.188:9092"}, config)
	if err != nil {
		panic(err)
	}
	defer consumer.Close()
	//根据消费者获取指定的主题分区的消费者,Offset为0为获取最新的消息，默认设置数据库最大.offset
	partitionConsumer, err := consumer.ConsumePartition("CAR_NUMBER_GO", 0, offset)
	if err != nil {
		fmt.Println("error get partition consumer", err)
	}
	defer partitionConsumer.Close()
	//循环等待接受消息.
	for {
		select {
		//接收消息通道和错误通道的内容.
		case msg := <-partitionConsumer.Messages():
			//先检查是否存在再插入
			num := 0
			value := string(msg.Value)
			o.Raw("select count(1) num  from seckill where info =?", value).QueryRow(&num)
			if num == 0 {
				_, err := o.Raw("insert into seckill (date,info,offset) values (?,?,?)", time.Now(), value, msg.Offset).Exec()
				if err == nil {
					fmt.Println("mysql : ", value+"插入成功")
				}

			}
		case err := <-partitionConsumer.Errors():
			fmt.Println(err.Err)
		}
	}

}
