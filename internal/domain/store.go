package domain

// Store 持久化生成快照。
type Store interface {
	Save(rec Record) (int64, error)
	Find(id int64) (Record, error)
	List(offset, limit int) ([]RecordSummary, error)
	Count() (int, error)
	Ping() error
	Close() error
}
