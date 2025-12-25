package scylladb

import "github.com/gocql/gocql"

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

	return &ScyllaDB{
		Session: session,
	}, nil
}

func (s *ScyllaDB) Close() {
	if s.Session != nil {
		s.Session.Close()
	}
}
