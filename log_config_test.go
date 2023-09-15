package logrpcs

import (
	"testing"
	"time"
)

func Test_timeTrack(t *testing.T) {

	now := time.Now()
	for i := 0; i < 60; i++ {
		tt := now.Add(time.Minute * time.Duration(i))
		ii := tt.Truncate(5 * time.Minute)
		t.Log(tt.Format("2006-01-02 15:04:05"), ii.Format("2006-01-02 15:04:05"))
	}

}

func Benchmark_GzWrite(b *testing.B) {

	mytopic := Topic{}
	mytopic.Name = "test"
	mytopic.ExtName = ".log"
	mytopic.S3 = "test"
	mytopic.TimeCheck = "1s"
	mytopic.MaxLine = 10000000
	mytopic.UpType = 2
	mytopic.maxTime = time.Duration(5 * time.Minute)
	mytopic.cache = make(chan []byte, 100000000) //chan 不能再独立携程中创建
	mytopic.exit = make(chan bool, 1)
	// mytopic.rwL = sync.RWMutex{}
	go mytopic.Init()
	b.ResetTimer()

	// gf, _ := NewGZip("test.gz")

	for i := 0; i < b.N; i++ {

		for i := 0; i < 1000; i++ {
			mytopic.WriteToCache([]byte("hello world"))
			// mytopic.Write(func(w io.Writer) error {
			// 	_, err := w.Write([]byte("hello world"))
			// 	return err
			// })
			// mytopic.WriteToCache([]byte("hello world"))

		}
	}
	close(mytopic.cache)

	mytopic.Done()
}
