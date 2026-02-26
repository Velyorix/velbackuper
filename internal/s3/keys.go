package s3

import (
	"path"
	"strings"
)

const (
	ArchivesPrefix  = "archives"
	ManifestsPrefix = "manifests"
	LatestPrefix    = "latest"
	ObjectsPrefix   = "objects"
	SnapshotsPrefix = "snapshots"
	IndexesPrefix   = "indexes"
	LocksPrefix     = "locks"
)

func ArchiveObjectKey(job, yyyy, mm, dd, filename string) string {
	return path.Join(ArchivesPrefix, job, yyyy, mm, dd, filename)
}

func ManifestKey(job, timestamp string) string {
	return path.Join(ManifestsPrefix, job, timestamp+".json")
}

func LatestKey(job string) string {
	return path.Join(LatestPrefix, job+".json")
}

func SnapshotKey(job, timestamp string) string {
	return path.Join(SnapshotsPrefix, job, timestamp+".json")
}

func IndexKey(job, timestamp string) string {
	return path.Join(IndexesPrefix, job, timestamp+".json")
}

func ObjectKey(hashPrefix, hash string) string {
	return path.Join(ObjectsPrefix, hashPrefix, hash)
}

func LockKey(job string) string {
	return path.Join(LocksPrefix, job+".lock")
}

func ParseArchiveKey(relativeKey string) (job, yyyy, mm, dd, filename string) {
	relativeKey = strings.Trim(relativeKey, "/")
	parts := strings.Split(relativeKey, "/")
	if len(parts) < 6 || parts[0] != ArchivesPrefix {
		return "", "", "", "", ""
	}
	job = parts[1]
	yyyy = parts[2]
	mm = parts[3]
	dd = parts[4]
	filename = strings.Join(parts[5:], "/")
	return job, yyyy, mm, dd, filename
}

func SnapshotsPrefixForJob(job string) string {
	return path.Join(SnapshotsPrefix, job) + "/"
}

func ArchivesPrefixForJob(job string) string {
	return path.Join(ArchivesPrefix, job) + "/"
}
