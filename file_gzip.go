package logrpcs

import (
	"compress/gzip"
	"errors"
	"os"

	"github.com/sirupsen/logrus"
)

func NewGZip(fileName string) (gf *GzipFile, err error) {
	gf = new(GzipFile)
	err = gf.Open(fileName)
	logrus.Info("new file:", fileName)

	return gf, err
}

// gzip 压缩， 非线程安全
type GzipFile struct {
	fw       *os.File     // 磁盘文件连接句柄
	gw       *gzip.Writer // 数据压缩句柄
	fileName string
}

func (gf *GzipFile) IsNil() bool {
	return gf.fw == nil && gf.gw == nil
}
func (gf *GzipFile) Open(filePath string) (err error) {
	gf.fileName = filePath
	gf.fw, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0775)
	if err != nil {
		return
	}
	gf.gw = gzip.NewWriter(gf.fw)
	if gf.gw == nil {
		err = errors.New("gzip create is error ")
	}
	return
}
func (gf *GzipFile) GetFileName() string {
	return gf.fileName
}
func (gf *GzipFile) Write(line []byte) (int, error) {
	return gf.gw.Write(line)
}

//计算尺寸
func (gf *GzipFile) Size() (int64, error) {
	gf.Flush()
	info, err := gf.fw.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

//将gzip 压缩流向磁盘写入一次。
// gzip自带一个flush 函数， 该函数会造成想磁盘中声明一个占位符的问题。 最终大小和实际大小可能会不符.
// 且gzip的flush 提示主要用于网络传输
func (gf *GzipFile) Flush() error {
	if gf.gw != nil {
		err := gf.gw.Close()
		if err != nil {
			logrus.Error(err)
		}
	}

	gf.gw = nil
	gf.gw = gzip.NewWriter(gf.fw)
	return nil
}

// 文件close 操作会去掉 磁盘文件的。tmp扩展名
func (gf *GzipFile) Close() (err error) {

	err = gf.gw.Close()
	if err != nil {
		return err
	}
	err = gf.fw.Close()
	if err != nil {
		return err
	}
	gf.fw = nil
	gf.gw = nil

	return err
}
