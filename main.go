package main

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/astaxie/beego"
	_ "miaosha-go/routers"
	"sync"
)
var (
	wg  sync.WaitGroup
)
func main() {
	//注册Kafka消费者
	go KafkaConsumer()
	beego.Run()

}

func KafkaConsumer()  {

	config := sarama.NewConfig()
	//接收失败通知
	config.Consumer.Return.Errors = true
	// 根据给定的代理地址和配置创建一个消费者
	consumer, err := sarama.NewConsumer([]string{"47.105.36.188:9092"}, config)
	if err != nil {
		panic(err)
	}
	defer consumer.Close()

	//根据消费者获取指定的主题分区的消费者,Offset这里指定为获取最新的消息.
	partitionConsumer, err := consumer.ConsumePartition("CAR_NUMBER", 0, 0)
	if err != nil {
		fmt.Println("error get partition consumer", err)
	}
	defer partitionConsumer.Close()
	//循环等待接受消息.
	for {
		select {
		//接收消息通道和错误通道的内容.
		case msg := <-partitionConsumer.Messages():
			fmt.Println("msg offset: ", msg.Offset, " partition: ", msg.Partition, " timestrap: ", msg.Timestamp.Format("2006-Jan-02 15:04"), " value: ", string(msg.Value))
		case err := <-partitionConsumer.Errors():
			fmt.Println(err.Err)
		}
	}

}
