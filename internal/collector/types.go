package collector

type SourceItem struct {
	Path string
	Size int64
}

const (
	CollectorMySQL      = "mysql"
	CollectorFilesystem = "filesystem"
	CollectorPresets    = "presets"
)
