package logrpcs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"webapi_go/app"
	"webapi_go/configure"
	"webapi_go/utils"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/sirupsen/logrus"
)

var (
	YamlConfigure = new(TopicsConf)
	TopicsStop    = sync.WaitGroup{}
	hostname      string
)

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = utils.GetRandomString2(15)
	}
	// logrus.SetOutput(io.Stderr)
	logrus.SetOutput(os.Stderr)
	app.OnStart(func() {
		TopicInit()
	})
	app.OnExit(func() {
		TopicClose()
		// TopicsStop.Wait()
	})

}

func TopicClose() {
	for _, v := range YamlConfigure.Topics {
		v.Close()
		v.Done()
	}
}

func TopicInit() {
	err := json.Unmarshal(configure.Config.S3Uploader, &YamlConfigure)
	if err != nil {
		logrus.Error(err)
		return
	}

	for name, ss := range YamlConfigure.Topics {
		ss.Name = name
		ss.maxTime, err = time.ParseDuration(ss.TimeCheck)
		ss.cache = make(chan []byte, 10000)
		ss.exit = make(chan bool, 1)
		if err != nil {
			logrus.Error(err)
		}
		go ss.Init()
		// Topics[ss.Name] = &YamlConfigure.Topics[i]
		// Topics[ss.Name].rwL = sync.RWMutex{}
		// ss.rwL = sync.RWMutex{}
		//Topics[ss.Name].Init()
		//Topics[ss.Name].Chan = make(chan *bytes.Buffer)

		//go Topics[ss.Name].Run()
	}

}

func WriteJsonToTopic(topic string, obj interface{}) {
	buf := bytes.NewBuffer(nil)
	json.NewEncoder(buf).Encode(obj)
	WriteToTopic(topic, "", buf)
}

func WriteToTopic(topic string, child string, r io.Reader) {
	if t, ok := YamlConfigure.Topics[topic]; ok {
		if buf, ok := r.(*bytes.Buffer); ok {
			t.WriteToCache(buf.Bytes())
		} else if buf, err := io.ReadAll(r); err == nil {
			t.WriteToCache(buf)
		}
	} else {
		logrus.Error("Topic ", topic, " not exists!!")
	}
}

type Topic struct {
	Name      string `json:"name" db:"name"`
	S3        string `json:"s3" db:"s3"`             //s3存储路径和本地存储路径格式保持一致，便于维护
	MaxSize   int64  `json:"max_size" db:"max_size"` //最大文件尺寸
	MaxLine   int    `json:"max_line_to_flush" db:"max_line"`
	TimeCheck string `json:"time_check" db:"time_check"` //最长写入时长
	ExtName   string `json:"ext_name,omitempty"`         //文件扩展名
	maxTime   time.Duration
	gf        FileEngine  //压缩文件句柄
	cache     chan []byte //
	exit      chan bool
	wLine     int    //当前写入行， 写入 1000行落一次磁盘
	fName     string // 生成基于时间的日志文件名
	fPath     string // 生成基于时间的日志路径

	// signalClose chan bool
	UpType   int    `json:"up_type,omitempty"`   //上传方式 ， 默认0 使用上传服务器上传, 1程序自己上传 , 2 不上传， 留存在本地
	UpSAddr  string `json:"up_server,omitempty"` // 上传服务器地址，  为空的时候， 使用 config.ini 中的默认服务器配置
	UpRegion string `json:"up_region,omitempty"` // 当 up_type 1 , 表示s3桶的可用区位置
	Baseurl  string `json:"base_url,omitempty"`  // 默认空， 不为空时，可以指定一个独立的文件夹, 如果 UpType =0 时， 默认文件夹不在EFS中， 会造成 up_server 上传失败
}

func (tc *Topic) initOpen() error {
	tc.wLine = 0
	var err error
	if tc.ExtName == "" {
		tc.fName = tc.Name + "-%s-2006-01-02-15-04-05.gz.tmp"
		tc.fPath = YamlConfigure.LocalFile + tc.S3
		tc.gf, err = NewGZip(tc.makeFile())
	} else if tc.ExtName == ".log" {
		tc.fName = tc.Name + "-%s-2006-01-02-15-04-05.log.tmp"
		tc.fPath = YamlConfigure.LocalFile + tc.S3
		tc.gf, err = NewLogFile(tc.makeFile())
	}
	return err
}
func (tc *Topic) Init() {
	defer func() {
		tc.exit <- true
	}()
	tc.initOpen()
	tick := time.NewTicker(tc.maxTime)
	// logrus.Info("xxxx")
	for {
		select {
		case buf, ok := <-tc.cache:
			if !ok {
				return
			}
			tc.gf.Write(buf)

			tc.wLine++
			if tc.wLine >= tc.MaxLine {
				tc.stop()
				tc.initOpen()
			}
		case <-tick.C:
			tc.stop()
			tc.initOpen()
			tick.Reset(tc.maxTime)

		}
	}

	// go tc.Run()
}
func (tc *Topic) Done() {
	<-tc.exit
	if tc.gf != nil {
		tc.stop()
	}
	logrus.Info("Topic ", tc.Name, " Done")
}

func (tc *Topic) getServerName() string {
	return hostname
}

func (tc *Topic) makeFile() string {
	now := time.Now().Truncate(tc.maxTime)
	p := tc.fPath + now.Format("2006/01/02/15/")
	n := now.Format(tc.fName)
	err := os.MkdirAll(p, 0775)
	if err != nil {
		logrus.Error(err.Error() + " makefile")

	}
	return fmt.Sprintf(p+n, tc.getServerName())
}

// 清空信息
func (tc *Topic) stop() error {
	// if tc.timeCron != nil {
	// 	tc.timeCron.Stop() //关闭时间控制
	// 	tc.timeCron = nil
	// }
	tc.wLine = 0
	if tc.gf != nil {
		tc.gf.Close()

		fileName := tc.gf.GetFileName()
		if size, err := tc.gf.Size(); err == nil {
			if size > 0 { // 大文件上传
				//去tmp字样
				if strings.Index(fileName, ".tmp") > -1 {
					os.Rename(fileName, fileName[:len(fileName)-4])
					fileName = fileName[:len(fileName)-4]
				}
				// err := os.Rename(fileName, fileName[:len(fileName)-4])
				// //err = tc.gf.Open(tc.fPath, tc.fName)
				// log.Println("rename", fileName, "to", fileName[:len(fileName)-4], err)
				tc.uploadToServer(fileName)
			} else { // 空文件删除
				os.Remove(fileName)
			}
		}
	}

	tc.gf = nil

	return nil
}

func (tc *Topic) Close() error {
	close(tc.cache)
	return nil
}
func (tc *Topic) WriteToCache(buf []byte) {
	tc.cache <- buf
}

// func (tc *Topic) Write(write func(io.Writer) error) {

// 	err := write(tc.gf)
// 	//_, err := tc.gf.Write(buf)
// 	if err != nil {
// 		logrus.Error(err)
// 	}

// }

func (tc *Topic) uploadToServer(fileName string) {
	//上传文件方案处理
	switch tc.UpType {
	case 0:
		req, _ := http.NewRequest("POST", tc.UpSAddr, bytes.NewBufferString(fileName))
		_, err := http.DefaultClient.Do(req)
		if err != nil {
			// notify.SendToDing("57b833298068bbae03df3f4f8675f0fd8b01df38416752c702462cb2a97a9062", &notify.Body{
			// 	Markdown: notify.Markdown{
			// 		Title: "日志上传异常",
			// 		Text:  logserver_uri + " \n 上传异常: " + err.Error() + " \n上传服务:" + tc.UpSAddr + "\nLogServer:" + tc.getServerName(),
			// 	},
			// 	At: notify.At{
			// 		IsAtAll: true,
			// 	},
			// })
			logrus.Error("post:", err)
		}
	case 1:
		tc.uploadToS3(fileName)
	case 2:
	}
}

func (tc *Topic) uploadToS3(fileName string) {
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0600)
	if err != nil {
		// notify.SendToDing("57b833298068bbae03df3f4f8675f0fd8b01df38416752c702462cb2a97a9062", &notify.Body{
		// 	Markdown: notify.Markdown{
		// 		Title: "日志上传异常",
		// 		Text:  fileName + " \n 上传异常: " + err.Error() + " \n上传服务: s3" + "\nLogServer:" + tc.getServerName(),
		// 	},
		// 	At: notify.At{
		// 		IsAtAll: true,
		// 	},
		// })
		logrus.Error("ERROR", err)
		return
	}
	defer f.Close()
	base_len := len(YamlConfigure.LocalFile)
	short_url := fileName[base_len:]
	pathDir := strings.Split(short_url, "/")
	if len(pathDir) > 0 {
		res, err := s3client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(pathDir[0]),
			Key:         aws.String(strings.Join(pathDir[1:], "/")),
			Body:        f,
			ContentType: aws.String("binary/octet-stream"),
		})
		if err != nil {
			// notify.SendToDing("57b833298068bbae03df3f4f8675f0fd8b01df38416752c702462cb2a97a9062", &notify.Body{
			// 	Markdown: notify.Markdown{
			// 		Title: "日志上传异常",
			// 		Text:  fileName + " \n 上传异常: " + err.Error() + " \n上传服务: s3",
			// 	},
			// 	At: notify.At{
			// 		IsAtAll: true,
			// 	},
			// })
			logrus.Error(err)
			return

		} else {
			log.Println(res)
		}

		os.Remove(fileName)
		log.Println("Success")
	}
}
