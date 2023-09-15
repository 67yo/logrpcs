package logrpcs

type TopicsConf struct {
	LocalFile string            `json:"s3_local_path"`
	Topics    map[string]*Topic `json:"topics"`
}
