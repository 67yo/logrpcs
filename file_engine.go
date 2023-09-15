package logrpcs

type FileEngine interface {
	IsNil() bool                // 确认文件是否打开状态
	Open(filePath string) error //进入打开状态
	Write([]byte) (int, error)
	Size() (int64, error) //获取尺寸
	Flush() error         //落磁盘
	Close() error         //关闭文件
	GetFileName() string  // 输出当前文件路径
}
