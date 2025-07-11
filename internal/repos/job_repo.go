package repos

import (
	"database/sql"
	"sync"

	"github.com/Craig-Turley/task-scheduler.git/pkg/common/job"
	"github.com/bwmarrin/snowflake"
	_ "github.com/mattn/go-sqlite3"
)

type JobRepo interface {
	CreateJob(job *job.Job) error
	GetJob(id snowflake.ID) (*job.Job, error)
}

type SqliteJobRepo struct {
	store   *sql.DB
	mu      sync.Mutex
	cdcChan chan *job.Job // not sure about this. supposed to mimic cdc but not sure if I want/need it for my 0 users
}

func NewSqliteJobRepo(db *sql.DB) *SqliteJobRepo {
	return &SqliteJobRepo{
		store:   db,
		mu:      sync.Mutex{},
		cdcChan: make(chan *job.Job),
	}
}

// CreateJob sets the id of the job passed in from autoincrementing table and returns an erro
func (s *SqliteJobRepo) CreateJob(job *job.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := "INSERT INTO jobs (id, name, cron, retry_limit, type) VALUES (?, ?, ?, ?, ?)"
	_, err := s.store.Exec(query, job.Id, job.Name, job.Cron, job.RetryLimit, job.Type)
	if err != nil {
		return err
	}

	// again, this is an optimization for approxomatley 0 users
	// and if this deadlocks...we're screwed
	// deadlock counter: 2
	// "why do I even have this?" counter: 2
	// s.cdcChan <- job

	return nil
}

// ahh yes, when this has a lot of history than say goodbye to your memory
func (s *SqliteJobRepo) GetJob(id snowflake.ID) (*job.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row := s.store.QueryRow("SELECT * FROM jobs WHERE id=?", id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	var job job.Job
	if err := row.Scan(&job.Id, &job.Name, &job.Cron, &job.RetryLimit, &job.Type); err != nil {
		return nil, err
	}

	return &job, nil
}
