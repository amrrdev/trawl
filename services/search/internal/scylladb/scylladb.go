package scylladb

import (
	"log"

	"github.com/gocql/gocql"
)

type ScyllaDB struct {
	Session *gocql.Session
}

func Connect(hosts ...string) (*ScyllaDB, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = "searchflow"
	cluster.Consistency = gocql.One

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	scylla := &ScyllaDB{
		Session: session,
	}

	if err := scylla.createTables(); err != nil {
		log.Printf("Warning: Failed to create tables: %v", err)
	}

	return scylla, nil
}

func (s *ScyllaDB) createTables() error {
	keyspaceQuery := `
		CREATE KEYSPACE IF NOT EXISTS searchflow
		WITH REPLICATION = {
			'class': 'SimpleStrategy',
			'replication_factor': 1
		}
	`
	if err := s.Session.Query(keyspaceQuery).Exec(); err != nil {
		return err
	}

	invertedIndexQuery := `
		CREATE TABLE IF NOT EXISTS searchflow.inverted_index (
			word text,
			doc_id uuid,
			term_frequency int,
			positions list<int>,
			PRIMARY KEY (word, doc_id)
		)
	`
	if err := s.Session.Query(invertedIndexQuery).Exec(); err != nil {
		return err
	}

	documentsQuery := `
		CREATE TABLE IF NOT EXISTS searchflow.documents (
			doc_id uuid PRIMARY KEY,
			title text,
			author text,
			file_path text,
			created_at timestamp
		)
	`
	if err := s.Session.Query(documentsQuery).Exec(); err != nil {
		return err
	}

	wordStatsQuery := `
		CREATE TABLE IF NOT EXISTS searchflow.word_stats (
			word text PRIMARY KEY,
			doc_count counter,
			total_occurrences counter
		)
	`
	if err := s.Session.Query(wordStatsQuery).Exec(); err != nil {
		return err
	}

	log.Println("✓ ScyllaDB tables created/verified")
	return nil
}

func (s *ScyllaDB) Close() {
	if s.Session != nil {
		s.Session.Close()
	}
}
