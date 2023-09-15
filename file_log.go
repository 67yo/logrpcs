package logrpcs

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

/*
type FileEngine interface {
	IsNil() bool                // 确认文件是否打开状态
	Open(filePath string) error //进入打开状态
	Write([]byte) (int, error)
	Size() (int64, error) //获取尺寸
	Flush() error         //落磁盘
	Close() error         //关闭文件
	GetFileName() string  // 输出当前文件路径
}

*/
func NewLogFile(fileName string) (fl *FileLog, err error) {
	fl = new(FileLog)
	err = fl.Open(fileName)
	logrus.Info("new file:", fileName)

	return fl, err
}

type FileLog struct {
	fw       *os.File
	fileName string
}

func (fl *FileLog) IsNil() bool {
	return fl.fw == nil
}
func (fl *FileLog) Open(filePath string) (err error) {
	fl.fileName = filePath
	fl.fw, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0775)
	return
}
func (fl *FileLog) Write(buf []byte) (int, error) {
	l := len(buf)
	n, err := fl.fw.Write(buf)
	if n != l && err == nil {
		return n, errors.New(fmt.Sprintf("write len error , has %d ,but is %d", l, n))
	}
	return n, err
}
func (fl *FileLog) Size() (int64, error) {
	ff, err := os.Stat(fl.fileName)
	if err != nil {
		return 0, err
	}
	return ff.Size(), nil
}
func (fl *FileLog) Flush() error {
	return nil
}
func (fl *FileLog) Close() error {
	if fl.fw != nil {
		err := fl.fw.Close()
		fl.fw = nil
		return err
	}
	return nil
}
func (fl *FileLog) GetFileName() string {
	return fl.fileName
}
