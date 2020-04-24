package main

type Relationship struct {
	Id1       uint64 `json:"id1"`
	Id2       uint64 `json:"id2"`
	Alias     string `json:"alias"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	DeletedAt int64  `json:"deleted_at"`
}
